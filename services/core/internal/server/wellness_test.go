package server

import (
	"context"
	pb "github.com/architsangal/AteThat/services/core/internal/gen/atethat/v1"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"os"
	"testing"
	"time"
)

func phaseTwoServer(t *testing.T) (*Server, context.Context, context.Context) {
	t.Helper()
	uri := os.Getenv("ATETHAT_TEST_MONGO_URI")
	if uri == "" {
		t.Skip("local replica set required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatal(err)
	}
	db := client.Database("atethat_phase2_test_" + primitive.NewObjectID().Hex())
	t.Cleanup(func() { db.Drop(context.Background()); client.Disconnect(context.Background()) })
	s := &Server{DB: db, Client: client}
	user := primitive.NewObjectID()
	if _, err := db.Collection("users").InsertOne(ctx, userDoc{ID: user, EmailDisplay: "local-qa@example.invalid"}); err != nil {
		t.Fatal(err)
	}
	owner := context.WithValue(ctx, principalKey{}, user)
	other := context.WithValue(ctx, principalKey{}, primitive.NewObjectID())
	if _, err := db.Collection("user_profiles").InsertOne(ctx, profileDoc{UserID: user, Timezone: "Asia/Kolkata"}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Collection("user_preferences").InsertOne(ctx, preferencesDoc{UserID: user, DailyResetTime: "04:00"}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Collection("wellness_settings").Indexes().CreateOne(ctx, mongo.IndexModel{Keys: bson.D{{Key: "user_id", Value: 1}}, Options: options.Index().SetUnique(true)}); err != nil {
		t.Fatal(err)
	}
	return s, owner, other
}

func TestWellnessInputValidation(t *testing.T) {
	for _, tc := range []struct{ kind, amount, unit, want string }{{"WATER", "1.25", "L", "1250"}, {"WEIGHT", "100", "lb", "45.359237"}, {"WATER", "250", "ml", "250"}} {
		input := &pb.WellnessRecord{Kind: tc.kind, Amount: tc.amount, Unit: tc.unit, LoggedAt: time.Now().UTC().Format(time.RFC3339)}
		d, err := wellnessInput(input)
		if err != nil || d.Amount.String() != tc.want {
			t.Fatalf("%+v: %+v %v", tc, d, err)
		}
	}
	for _, input := range []*pb.WellnessRecord{
		{Kind: "WATER", Amount: "-1", Unit: "ml"}, {Kind: "WATER", Amount: "0", Unit: "ml"}, {Kind: "WATER", Amount: "1e3", Unit: "ml"},
		{Kind: "WATER", Amount: "1", Unit: "kg"}, {Kind: "WEIGHT", Amount: "1", Unit: "ml"}, {Kind: "UNKNOWN", Amount: "1", Unit: "ml"},
		{Kind: "WATER", Amount: "1", Unit: "ml", Intensity: "LIGHT"}, {Kind: "ACTIVITY", Amount: "10", Unit: "min"},
		{Kind: "ACTIVITY", Amount: "10", Unit: "min", Name: "Walk", Intensity: "LIGHT", EstimatedKcal: "-2"},
	} {
		input.LoggedAt = time.Now().UTC().Format(time.RFC3339)
		if _, err := wellnessInput(input); status.Code(err) != codes.InvalidArgument {
			t.Fatalf("accepted invalid input: %+v %v", input, err)
		}
	}
}

func TestWellnessOwnershipVersionsAndTrackingDayIntegration(t *testing.T) {
	s, owner, other := phaseTwoServer(t)
	// 01:00 India belongs to the previous Tracking Day with a 04:00 reset.
	yesterday := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	input := &pb.WellnessRecord{Kind: "WATER", Amount: "0.5", Unit: "L", LoggedAt: yesterday + "T01:00:00+05:30"}
	saved, err := s.SaveRecord(owner, input)
	if err != nil {
		t.Fatal(err)
	}
	date, _ := time.Parse("2006-01-02", yesterday)
	wantDay := date.AddDate(0, 0, -1).Format("2006-01-02")
	if saved.Amount != "500" || saved.Unit != "ml" || saved.TrackingDay != wantDay {
		t.Fatalf("bad normalization: %+v", saved)
	}
	query := &pb.WellnessListRequest{FromTrackingDay: wantDay, ToTrackingDay: yesterday, Limit: 1}
	page, err := s.ListRecords(other, query)
	if err != nil || len(page.Items) != 0 {
		t.Fatal("foreign records visible", err)
	}
	if _, err := s.DeleteRecord(other, &pb.WellnessDeleteRequest{Id: saved.Id, Version: 1}); status.Code(err) != codes.Aborted {
		t.Fatal("foreign deletion accepted", err)
	}
	user, _ := Principal(owner)
	if _, err := s.DB.Collection("user_profiles").UpdateOne(owner, bson.M{"user_id": user}, bson.M{"$set": bson.M{"timezone": "UTC"}}); err != nil {
		t.Fatal(err)
	}
	saved.Amount = "600"
	updated, err := s.SaveRecord(owner, saved)
	if err != nil || updated.TrackingDay != wantDay || updated.Timezone != "Asia/Kolkata" || updated.Version != 2 {
		t.Fatal("historical context not preserved", updated, err)
	}
	if _, err := s.SaveRecord(owner, saved); status.Code(err) != codes.Aborted {
		t.Fatal("stale edit accepted", err)
	}
	input.LoggedAt = yesterday + "T02:00:00Z"
	if _, err := s.SaveRecord(owner, input); err != nil {
		t.Fatal(err)
	}
	page, err = s.ListRecords(owner, query)
	if err != nil || len(page.Items) != 1 || page.NextCursor == "" {
		t.Fatal("missing pagination", page, err)
	}
	query.Cursor = page.NextCursor
	page, err = s.ListRecords(owner, query)
	if err != nil || len(page.Items) != 1 || page.NextCursor != "" {
		t.Fatal("pagination skipped records", page, err)
	}
	if _, err := s.DeleteRecord(owner, &pb.WellnessDeleteRequest{Id: updated.Id, Version: updated.Version}); err != nil {
		t.Fatal(err)
	}
	query.Cursor = ""
	page, err = s.ListRecords(owner, query)
	if err != nil || len(page.Items) != 1 || page.Items[0].Id == updated.Id {
		t.Fatal("deleted record remains", page, err)
	}
	prefs, err := s.SaveSettings(owner, &pb.WellnessSettings{WaterTargetMl: "2000"})
	if err != nil || prefs.Version != 1 {
		t.Fatal(prefs, err)
	}
	if _, err := s.SaveSettings(owner, &pb.WellnessSettings{WaterTargetMl: "3000"}); status.Code(err) != codes.Aborted {
		t.Fatal("stale settings accepted", err)
	}
	if prefs, err = s.GetSettings(other, &pb.FoodEmpty{}); err != nil || prefs.WaterTargetMl != "" {
		t.Fatal("foreign goal visible", err)
	}
	if _, err := s.SaveSettings(owner, &pb.WellnessSettings{Version: 1}); err != nil {
		t.Fatal("clear goal failed", err)
	}
}
