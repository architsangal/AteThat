package server

import (
	"strings"
	"testing"
	"time"

	pb "github.com/architsangal/AteThat/services/core/internal/gen/atethat/v1"
	"go.mongodb.org/mongo-driver/bson"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestTrackedNutrientsValidateSelection(t *testing.T) {
	if err := validateTrackedNutrients(trackedNutrientCodes); err != nil {
		t.Fatal("select-all rejected", err)
	}
	for _, selection := range [][]string{nil, {"PROTEIN"}, {"ENERGY", "ENERGY"}, {"ENERGY", "MADE_UP"}} {
		if validateTrackedNutrients(selection) == nil {
			t.Fatal("invalid selection accepted", selection)
		}
	}
}

func TestManualFoodSnapshotsAndTrackedPresentation(t *testing.T) {
	s, owner, other := phaseTwoServer(t)
	user, _ := Principal(owner)
	if _, err := s.DB.Collection("user_preferences").UpdateOne(owner, bson.M{"user_id": user}, bson.M{"$set": bson.M{"version": 1, "dashboard_nutrients": []string{"ENERGY", "PROTEIN"}}}); err != nil {
		t.Fatal(err)
	}
	profile, err := s.GetProfile(owner, &pb.Empty{})
	if err != nil || profile.Email != "local-qa@example.invalid" {
		t.Fatal("account email unavailable", profile, err)
	}
	manual := &pb.UpsertCustomFoodRequest{Name: "One-off ramen", FoodState: "PREPARED", BaseQuantity: &pb.FoodQuantity{Amount: "100", Unit: "g"}, Nutrients: map[string]*pb.FoodNutrient{
		"ENERGY": {Status: "KNOWN", Amount: "200", Unit: "kcal"}, "PROTEIN": {Status: "KNOWN", Amount: "5", Unit: "g"}, "FAT": {Status: "KNOWN", Amount: "0", Unit: "g"}, "VITAMIN_C": {Status: "KNOWN", Amount: "12", Unit: "mg"},
	}}
	request := &pb.PreviewFoodGroupRequest{LoggedAt: time.Now().UTC().Format(time.RFC3339), MealCategory: "LUNCH", Items: []*pb.FoodLogItemInput{{Position: 0, FoodType: "MANUAL", ManualFood: manual, FoodState: "PREPARED", Quantity: &pb.FoodQuantity{Amount: "50", Unit: "g"}}}}
	preview, err := s.Preview(owner, request)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Group.AggregateFinalNutrition["ENERGY"].Amount != "100" || preview.Group.AggregateFinalNutrition["VITAMIN_C"].Amount != "6" {
		t.Fatal("manual scaling lost data", preview.Group)
	}
	if _, err = s.Confirm(other, &pb.ConfirmFoodGroupRequest{PreviewToken: preview.PreviewToken, IdempotencyKey: "foreign-manual-log"}); err == nil {
		t.Fatal("foreign confirmation allowed")
	}
	confirmed, err := s.Confirm(owner, &pb.ConfirmFoodGroupRequest{PreviewToken: preview.PreviewToken, IdempotencyKey: "manual-confirm-once"})
	if err != nil {
		t.Fatal(err)
	}
	replay, err := s.Confirm(owner, &pb.ConfirmFoodGroupRequest{PreviewToken: preview.PreviewToken, IdempotencyKey: "manual-confirm-once"})
	if err != nil || replay.Id != confirmed.Id {
		t.Fatal("confirmation duplicated", err)
	}
	count, err := s.DB.Collection("custom_foods").CountDocuments(owner, bson.M{"user_id": user})
	if err != nil || count != 0 {
		t.Fatal("one-time entry polluted library", count, err)
	}
	recent, err := s.ListUsedFoods(owner, &pb.ListUsedFoodsRequest{Sort: "RECENT", Limit: 20})
	if err != nil || len(recent.Items) != 0 {
		t.Fatal("manual entry offered as reusable reference", recent, err)
	}
	day := confirmed.TrackingDay
	summary, err := s.TextSummary(owner, &pb.TextSummaryRequest{FromTrackingDay: day, ToTrackingDay: day, Mode: "DAY", NutrientFilter: "ALL"})
	if err != nil || strings.Contains(summary.Text, "VITAMIN_C") || !strings.Contains(summary.Text, "PROTEIN") {
		t.Fatal("summary ignores selection", summary, err)
	}
	corrected, err := s.CorrectItem(owner, &pb.CorrectFoodLogItemRequest{Id: confirmed.Items[0].Id, Quantity: &pb.FoodQuantity{Amount: "100", Unit: "g"}, Reason: "Correct portion"})
	if err != nil || corrected.Group.AggregateFinalNutrition["VITAMIN_C"].Amount != "12" {
		t.Fatal("manual correction lost hidden nutrition", corrected, err)
	}
	request.Items[0].ManualFood = nil
	if _, err := s.Preview(owner, request); status.Code(err) != codes.InvalidArgument {
		t.Fatal("missing manual data accepted", err)
	}
}

func TestLegacyMealFractionalPortion(t *testing.T) {
	s, owner, _ := phaseTwoServer(t)
	food, err := s.CreateCustomFood(owner, &pb.UpsertCustomFoodRequest{Name: "Cooked dal", FoodState: "COOKED", BaseQuantity: &pb.FoodQuantity{Amount: "100", Unit: "g"}, Nutrients: map[string]*pb.FoodNutrient{"ENERGY": {Status: "KNOWN", Amount: "150", Unit: "kcal"}}})
	if err != nil {
		t.Fatal(err)
	}
	meal, err := s.CreateMeal(owner, &pb.UpsertMealRequest{Name: "Old saved meal", Components: []*pb.MealComponent{{Position: 0, FoodType: "CUSTOM", CustomFoodId: food.Id, FoodState: "COOKED", Quantity: &pb.FoodQuantity{Amount: "200", Unit: "g"}}}})
	if err != nil {
		t.Fatal(err)
	}
	preview, err := s.PreviewMeal(owner, &pb.PreviewMealRequest{MealId: meal.Id, Servings: "0.5", LoggedAt: time.Now().UTC().Format(time.RFC3339), MealCategory: "OTHER"})
	if err != nil || preview.Group.Items[0].Quantity.Amount != "100" || preview.Group.AggregateFinalNutrition["ENERGY"].Amount != "150" {
		t.Fatal("legacy meal lost fractional serving", preview, err)
	}
}
