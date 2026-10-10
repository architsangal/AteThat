package server

import (
	"context"
	"os"
	"testing"
	"time"

	pb "github.com/architsangal/AteThat/services/core/internal/gen/atethat/v1"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestRetentionIntegration(t *testing.T) {
	uri := os.Getenv("ATETHAT_TEST_MONGO_URI")
	if uri == "" {
		t.Skip("set ATETHAT_TEST_MONGO_URI to test retention against a local replica set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Disconnect(context.Background())
	name := "atethat_retention_test_" + primitive.NewObjectID().Hex()
	db := client.Database(name)
	defer db.Drop(context.Background())
	server := &Server{DB: db, Client: client}
	userID, groupID := primitive.NewObjectID(), primitive.NewObjectID()
	loggedAt := time.Now().UTC().AddDate(0, 0, -100).Truncate(time.Millisecond)
	confirmedAt := loggedAt.Add(time.Hour)
	energy, err := primitive.ParseDecimal128("123.45")
	if err != nil {
		t.Fatal(err)
	}
	group := foodLogGroupDoc{ID: groupID, UserID: userID, RootGroupID: groupID, Status: "CONFIRMED",
		TrackingDay: loggedAt.Format("2006-01-02"), TimezoneAtLogging: "UTC", DailyResetTimeAtLog: "00:00",
		LoggedAt: loggedAt, ConfirmedAt: confirmedAt, ItemCount: 1,
		Aggregate: map[string]foodNutrientDoc{"ENERGY": {Status: "KNOWN", Amount: &energy, Unit: "kcal"}}}
	if _, err := db.Collection("food_log_groups").InsertOne(ctx, group); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Collection("food_logs").InsertOne(ctx, foodLogItemDoc{ID: primitive.NewObjectID(), UserID: userID,
		GroupID: groupID, TrackingDay: group.TrackingDay, LoggedAt: loggedAt, ConfirmedAt: confirmedAt}); err != nil {
		t.Fatal(err)
	}
	if err := server.RunRetentionOnce(ctx); err != nil {
		t.Fatal(err)
	}
	var summary dailyNutritionSummaryDoc
	if err := db.Collection("daily_nutrition_summaries").FindOne(ctx, bson.M{"user_id": userID, "tracking_day": group.TrackingDay}).Decode(&summary); err != nil {
		t.Fatal(err)
	}
	if summary.ConfirmedGroupCount != 1 || summary.ConfirmedItemCount != 1 || summary.SchemaVersion != 1 ||
		!summary.EarliestLoggedAtUTC.Equal(loggedAt) || !summary.LatestLoggedAtUTC.Equal(loggedAt) ||
		summary.Aggregate["ENERGY"].Amount.String() != "123.45" {
		t.Fatalf("incorrect compacted summary: %+v", summary)
	}
	for _, collection := range []string{"food_log_groups", "food_logs"} {
		count, err := db.Collection(collection).CountDocuments(ctx, bson.M{"user_id": userID})
		if err != nil || count != 0 {
			t.Fatalf("%s detail remains: %d, %v", collection, count, err)
		}
	}
	if err := server.RunRetentionOnce(ctx); err != nil {
		t.Fatalf("second retention pass: %v", err)
	}
	userContext := context.WithValue(ctx, principalKey{}, userID)
	history, err := server.NutritionHistory(userContext, &pb.NutritionHistoryRequest{
		FromTrackingDay: group.TrackingDay, ToTrackingDay: group.TrackingDay, Limit: 10})
	if err != nil || len(history.Items) != 1 || !history.Items[0].DetailCompacted || history.Items[0].Totals["ENERGY"].Amount != "123.45" {
		t.Fatalf("compacted history unavailable: %+v %v", history, err)
	}
	analysis, err := server.FullAnalysis(userContext, &pb.FullAnalysisRequest{
		From: loggedAt.Add(-time.Hour).Format(time.RFC3339Nano), To: loggedAt.Add(time.Hour).Format(time.RFC3339Nano)})
	if err != nil || analysis.Totals["ENERGY"].Amount != "123.45" || analysis.CompactedDayCount != 1 {
		t.Fatalf("compacted analysis unavailable: %+v %v", analysis, err)
	}
}
