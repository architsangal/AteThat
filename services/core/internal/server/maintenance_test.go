package server

import (
	pb "github.com/architsangal/AteThat/services/core/internal/gen/atethat/v1"
	"go.mongodb.org/mongo-driver/bson"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"
	"testing"
	"time"
)

func TestMaintenanceHistoryAndWellnessDeficit(t *testing.T) {
	s, owner, other := phaseTwoServer(t)
	plan, err := s.GetMaintenance(owner, &pb.FoodEmpty{})
	if err != nil {
		t.Fatal(err)
	}
	if plan.SuggestedKcal != "" || plan.UnavailableReason == "" {
		t.Fatal("missing age invented", plan)
	}
	plan, err = s.SaveMaintenance(owner, &pb.MaintenanceSaveRequest{Version: plan.Version, Kcal: "2000", Mode: "MANUAL"})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.History) != 1 || plan.History[0].EffectiveFromTrackingDay != plan.TrackingDay {
		t.Fatal(plan)
	}
	if _, err = s.SaveMaintenance(owner, &pb.MaintenanceSaveRequest{Version: 0, Kcal: "2200", Mode: "MANUAL"}); status.Code(err) != codes.Aborted {
		t.Fatal("stale save accepted", err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	food := &pb.UpsertCustomFoodRequest{Name: "Daily energy test", FoodState: "PREPARED", BaseQuantity: &pb.FoodQuantity{Amount: "100", Unit: "g"}, Nutrients: map[string]*pb.FoodNutrient{"ENERGY": {Status: "KNOWN", Amount: "1800", Unit: "kcal"}}}
	preview, err := s.Preview(owner, &pb.PreviewFoodGroupRequest{LoggedAt: now, MealCategory: "LUNCH", Items: []*pb.FoodLogItemInput{{FoodType: "MANUAL", ManualFood: food, FoodState: "PREPARED", Quantity: &pb.FoodQuantity{Amount: "100", Unit: "g"}}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Confirm(owner, &pb.ConfirmFoodGroupRequest{PreviewToken: preview.PreviewToken, IdempotencyKey: "maintenance-food-confirm"}); err != nil {
		t.Fatal(err)
	}
	workout, err := s.SaveRecord(owner, &pb.WellnessRecord{Kind: "ACTIVITY", Name: "Cycling", Intensity: "MODERATE", Amount: "30", Unit: "min", LoggedAt: now, EstimatedKcal: "300"})
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range []string{"500", "250"} {
		if _, err = s.SaveRecord(owner, &pb.WellnessRecord{Kind: "WATER", Amount: v, Unit: "ml", LoggedAt: now}); err != nil {
			t.Fatal(err)
		}
	}
	req := &pb.FullAnalysisRequest{From: plan.TrackingDay, To: plan.TrackingDay}
	data, err := s.DailyWellness(owner, req)
	if err != nil {
		t.Fatal(err)
	}
	row := data.Items[0]
	if row.CalorieDeficit != "500.000000" || row.WaterMl != "750.000000" || row.WorkoutCount != 1 {
		t.Fatal(row)
	}
	report, err := s.TextSummary(owner, &pb.TextSummaryRequest{FromTrackingDay: plan.TrackingDay, ToTrackingDay: plan.TrackingDay, Mode: "DAY", NutrientFilter: "ENERGY"})
	if err != nil || !strings.Contains(report.GetText(), "signed deficit 500.000000 kcal") {
		t.Fatal("deficit missing from factual summary", report, err)
	}
	// Unknown workout expenditure is not guessed as zero, even with known food.
	workout.EstimatedKcal = ""
	workout, err = s.SaveRecord(owner, workout)
	if err != nil {
		t.Fatal(err)
	}
	data, err = s.DailyWellness(owner, req)
	if err != nil || data.Items[0].CalorieDeficit != "" || data.Items[0].WorkoutsWithoutCalories != 1 {
		t.Fatal(data, err)
	}
	if _, err = s.DeleteRecord(owner, &pb.WellnessDeleteRequest{Id: workout.Id, Version: workout.Version}); err != nil {
		t.Fatal(err)
	}
	data, err = s.DailyWellness(owner, req)
	if err != nil || data.Items[0].CalorieDeficit != "200.000000" {
		t.Fatal(data, err)
	}
	// Editing baseline preserves today's result and schedules the next day.
	plan, err = s.SaveMaintenance(owner, &pb.MaintenanceSaveRequest{Version: plan.Version, Kcal: "2200", Mode: "MANUAL"})
	if err != nil {
		t.Fatal(err)
	}
	today, _ := time.Parse("2006-01-02", plan.TrackingDay)
	tomorrow := today.AddDate(0, 0, 1).Format("2006-01-02")
	data, err = s.DailyWellness(owner, &pb.FullAnalysisRequest{From: today.AddDate(0, 0, -1).Format("2006-01-02"), To: tomorrow})
	if err != nil {
		t.Fatal(err)
	}
	if data.Items[0].MaintenanceKcal != "" || data.Items[0].CalorieDeficit != "" || data.Items[0].WaterMl != "" {
		t.Fatal("missing day became zero", data.Items[0])
	}
	if data.Items[1].MaintenanceKcal != "2000" || data.Items[1].CalorieDeficit != "200.000000" || data.Items[2].MaintenanceKcal != "2200" {
		t.Fatal("history rewritten", data)
	}
	foreign, err := s.DailyWellness(other, req)
	if err == nil && len(foreign.Items) > 0 && foreign.Items[0].FoodKcal != "" {
		t.Fatal("foreign records leaked")
	}
	// Water-goal updates cannot erase maintenance history; CAS version is shared.
	if _, err = s.SaveSettings(owner, &pb.WellnessSettings{Version: plan.Version, WaterTargetMl: "2000"}); err != nil {
		t.Fatal(err)
	}
	saved, err := s.GetMaintenance(owner, &pb.FoodEmpty{})
	if err != nil || len(saved.History) != 2 {
		t.Fatal(saved, err)
	}
	user, _ := Principal(owner)
	count, err := s.DB.Collection("nutrition_target_versions").CountDocuments(owner, bson.M{"user_id": user})
	if err != nil || count != 0 {
		t.Fatal("maintenance changed food targets", count, err)
	}
}

func TestDeficitSignedMath(t *testing.T) {
	for _, tc := range []struct{ m, w, f, want string }{{"2000", "300", "2100", "200.000000"}, {"2000", "0", "2400", "-400.000000"}, {"2000.25", "0.5", "2000.75", "0.000000"}} {
		got, err := deficitValue(tc.m, tc.w, tc.f)
		if err != nil || got != tc.want {
			t.Fatal(got, err)
		}
	}
	if _, err := deficitValue("-1", "0", "0"); err == nil {
		t.Fatal("negative maintenance accepted")
	}
}

func TestCalculatedMaintenanceUsesFreshProfileAndMinimalActivity(t *testing.T) {
	s, owner, _ := phaseTwoServer(t)
	user, _ := Principal(owner)
	if _, err := s.DB.Collection("user_profiles").UpdateOne(owner, bson.M{"user_id": user}, bson.M{"$set": bson.M{"version": int64(1)}}); err != nil {
		t.Fatal(err)
	}
	profile, err := s.GetProfile(owner, &pb.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	dob, sex, activity := "1990-01-01", "FEMALE", "VERY_ACTIVE"
	profile, err = s.UpdateProfile(owner, &pb.UpdateProfileRequest{Version: profile.Version, DateOfBirth: &dob, Sex: &sex, ActivityLevel: &activity})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := s.GetMaintenance(owner, &pb.FoodEmpty{})
	if err != nil || plan.SuggestedKcal == "" || !plan.Inputs.AssumedHeight || !plan.Inputs.AssumedWeight {
		t.Fatal(plan, err)
	}
	if _, err = s.SaveMaintenance(owner, &pb.MaintenanceSaveRequest{Version: plan.Version, Kcal: plan.SuggestedKcal, Mode: "CALCULATED", ProfileVersion: profile.Version - 1}); status.Code(err) != codes.Aborted {
		t.Fatal("stale calculated profile accepted", err)
	}
	if _, err = s.SaveMaintenance(owner, &pb.MaintenanceSaveRequest{Version: plan.Version, Kcal: "100", Mode: "CALCULATED", ProfileVersion: profile.Version}); status.Code(err) != codes.Aborted {
		t.Fatal("tampered estimate accepted", err)
	}
	saved, err := s.SaveMaintenance(owner, &pb.MaintenanceSaveRequest{Version: plan.Version, Kcal: plan.SuggestedKcal, Mode: "CALCULATED", ProfileVersion: profile.Version})
	if err != nil || saved.History[0].Mode != "CALCULATED" {
		t.Fatal(saved, err)
	}
	activity = "SEDENTARY"
	if _, err = s.UpdateProfile(owner, &pb.UpdateProfileRequest{Version: profile.Version, ActivityLevel: &activity}); err != nil {
		t.Fatal(err)
	}
	updated, err := s.GetMaintenance(owner, &pb.FoodEmpty{})
	if err != nil || updated.SuggestedKcal != plan.SuggestedKcal || updated.History[0].Kcal != plan.SuggestedKcal {
		t.Fatal("usual activity changed the fixed minimal-activity baseline", updated, err)
	}
}
