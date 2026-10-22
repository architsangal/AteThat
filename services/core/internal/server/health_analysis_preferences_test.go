package server

import (
	pb "github.com/architsangal/AteThat/services/core/internal/gen/atethat/v1"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"testing"
	"time"
)

func TestHealthAnalysisSelectionOwnershipPersistenceAndVersion(t *testing.T) {
	s, owner, other := phaseTwoServer(t)
	userID, _ := Principal(owner)
	_, err := s.DB.Collection("user_preferences").UpdateOne(owner, bson.M{"user_id": userID}, bson.M{"$set": bson.M{"version": 1, "unit_system": "METRIC", "dashboard_nutrients": []string{"ENERGY"}}})
	if err != nil {
		t.Fatal(err)
	}
	makeItem := func(foreign bool) *pb.HealthItem {
		ctx := owner
		if foreign {
			ctx = other
		}
		day := time.Now().UTC().Format("2006-01-02")
		item, e := s.CreateItem(ctx, &pb.UpsertHealthItemRequest{Type: "TASK", Name: "Same name", StartDate: day, Schedule: &pb.HealthSchedule{Timezone: "UTC", Frequency: "DAILY", Interval: 1, LocalTimes: []string{"08:00"}, EffectiveFromDate: day}})
		if e != nil {
			t.Fatal(e)
		}
		return item
	}
	a, b, foreign := makeItem(false), makeItem(false), makeItem(true)
	save := func(version uint64, ids []string) (*pb.Preferences, error) {
		return s.UpdatePreferences(owner, &pb.UpdatePreferencesRequest{Version: version, AnalysisHealthItems: &pb.HealthItemSelection{Ids: ids}})
	}
	for _, ids := range [][]string{{foreign.Id}, {primitive.NewObjectID().Hex()}, {"invalid"}, {a.Id, a.Id}, {a.Id, a.Id, a.Id, a.Id, a.Id, a.Id, a.Id, a.Id, a.Id, a.Id, a.Id, a.Id, a.Id}} {
		if _, e := save(1, ids); status.Code(e) != codes.InvalidArgument {
			t.Fatalf("accepted invalid selection %v: %v", ids, e)
		}
	}
	saved, err := save(1, []string{a.Id, b.Id})
	if err != nil || saved.Version != 2 || len(saved.AnalysisHealthItemIds) != 2 || saved.DailyResetTime != "04:00" || saved.UnitSystem != "METRIC" || len(saved.DashboardNutrients) != 1 {
		t.Fatal(saved, err)
	}
	if _, err = save(1, []string{}); status.Code(err) != codes.Aborted {
		t.Fatal("stale version accepted", err)
	}
	loaded, err := s.GetPreferences(owner, &pb.Empty{})
	if err != nil || len(loaded.AnalysisHealthItemIds) != 2 {
		t.Fatal("selection not persisted", loaded, err)
	}
	metric := "IMPERIAL"
	updated, err := s.UpdatePreferences(owner, &pb.UpdatePreferencesRequest{Version: 2, UnitSystem: &metric})
	if err != nil || len(updated.AnalysisHealthItemIds) != 2 {
		t.Fatal("unrelated save cleared selection", updated, err)
	}
	cleared, err := save(3, []string{})
	if err != nil || len(cleared.AnalysisHealthItemIds) != 0 {
		t.Fatal("empty selection not respected", cleared, err)
	}
	// The configuration endpoint must never alter the schedules it references.
	listed, err := s.ListItems(owner, &pb.ListHealthItemsRequest{Limit: 50})
	if err != nil || len(listed.Items) != 2 {
		t.Fatal(listed, err)
	}
}
