package server

import (
	"strings"
	"testing"
	"time"

	pb "github.com/architsangal/AteThat/services/core/internal/gen/atethat/v1"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestPlanningDefaultsAndConversions(t *testing.T) {
	now := time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC)
	defaults, e := planningProfile(profileDoc{}, now)
	if e != nil || defaults.HeightCM != "175" || defaults.WeightKG != "70.4375" || defaults.Activity != "MODERATE" || !defaults.AssumedHeight || defaults.Age != 0 || defaults.Sex != "" {
		t.Fatalf("defaults: %+v %v", defaults, e)
	}
	height, _ := primitive.ParseDecimal128("1.8")
	weight, _ := primitive.ParseDecimal128("200")
	actual, e := planningProfile(profileDoc{Height: &quantityDoc{Amount: height, Unit: "m"}, Weight: &quantityDoc{Amount: weight, Unit: "lb"}, DateOfBirth: "2000-10-22", ActivityLevel: "LIGHT"}, now)
	if e != nil || actual.HeightCM != "180" || actual.WeightKG != "90.718474" || actual.Age != 25 || actual.AssumedWeight || actual.AssumedHeight || actual.AssumedActivity {
		t.Fatalf("conversion: %+v %v", actual, e)
	}
	birthday, e := planningProfile(profileDoc{DateOfBirth: "2000-10-22", Timezone: "Asia/Kolkata"}, time.Date(2026, 10, 21, 20, 0, 0, 0, time.UTC))
	if e != nil || birthday.Age != 26 {
		t.Fatal("age must use account calendar date, not UTC or nutrition reset", birthday, e)
	}
}
func TestCalculatedResetIsPreviewBoundAndAtomic(t *testing.T) {
	s, owner, other := phaseTwoServer(t)
	user, _ := Principal(owner)
	_, e := s.DB.Collection("user_profiles").UpdateOne(owner, bson.M{"user_id": user}, bson.M{"$set": bson.M{"version": int64(1), "date_of_birth": "2000-01-01", "sex": "MALE"}})
	if e != nil {
		t.Fatal(e)
	}
	plan, e := s.Planning(owner, &pb.TargetPlanningRequest{})
	if e != nil || !plan.PersonalizedEnabled || plan.ProteinRate != "1.2" || !plan.Inputs.AssumedWeight {
		t.Fatal(plan, e)
	}
	selection, e := s.Current(owner, &pb.Empty{})
	if e != nil || selection.Active != nil {
		t.Fatal("preview mutated targets", selection, e)
	}
	req := &pb.ResetCalculatedTargetsRequest{ProfileVersion: 1, PreviewToken: plan.ResetPreviewToken}
	if _, e := s.ResetCalculated(owner, req); status.Code(e) != codes.InvalidArgument {
		t.Fatal("missing applicability confirmation", e)
	}
	req.AdultScopeConfirmed = true
	if _, e := s.ResetCalculated(other, req); e == nil {
		t.Fatal("foreign preview accepted")
	}
	first, e := s.ResetCalculated(owner, req)
	if e != nil {
		t.Fatal(e)
	}
	for key, expected := range plan.CalculatedDefaults {
		value := first.Targets[key]
		if value.ActiveValue != expected || value.Provenance != "CALCULATED_DEFAULT" || value.IsManualOverride {
			t.Fatal(key, value, expected)
		}
	}
	if first.TargetBasis != "PERSONALIZED_ESTIMATE" || first.InputSnapshot.WeightKg != "70.4375" {
		t.Fatal(first)
	}
	if _, e := s.ResetCalculated(owner, req); status.Code(e) != codes.Aborted {
		t.Fatal("stale target reset accepted", e)
	}
	req.ExpectedActiveId, req.ExpectedRevision = first.Id, first.Revision
	req.PreviewToken = "tampered"
	if _, e := s.ResetCalculated(owner, req); status.Code(e) != codes.Aborted {
		t.Fatal("tampered preview accepted", e)
	}
	req.PreviewToken = plan.ResetPreviewToken
	_, e = s.DB.Collection("user_profiles").UpdateOne(owner, bson.M{"user_id": user}, bson.M{"$set": bson.M{"version": int64(2), "activity_level": "ACTIVE"}})
	if e != nil {
		t.Fatal(e)
	}
	if _, e := s.ResetCalculated(owner, req); status.Code(e) != codes.Aborted {
		t.Fatal("stale profile reset accepted", e)
	}
	newer, e := s.Planning(owner, &pb.TargetPlanningRequest{})
	if e != nil || newer.ProteinRate != "1.6" || newer.ResetPreviewToken == plan.ResetPreviewToken {
		t.Fatal(newer, e)
	}
	req.ProfileVersion, req.PreviewToken = 2, newer.ResetPreviewToken
	second, e := s.ResetCalculated(owner, req)
	if e != nil || second.Revision != first.Revision+1 {
		t.Fatal(second, e)
	}
	if second.Targets["PROTEIN"].ActiveValue == first.Targets["PROTEIN"].ActiveValue {
		t.Fatal("activity did not select protein rate")
	}
	var profile profileDoc
	if e := s.DB.Collection("user_profiles").FindOne(owner, bson.M{"user_id": user}).Decode(&profile); e != nil || profile.Weight != nil || profile.Height != nil {
		t.Fatal("assumptions persisted as measurements", profile, e)
	}
	var old targetVersionDoc
	id, _ := primitive.ObjectIDFromHex(first.Id)
	if e := s.DB.Collection("nutrition_target_versions").FindOne(owner, bson.M{"_id": id}).Decode(&old); e != nil || old.Status != "SUPERSEDED" || old.Targets["PROTEIN"].ActiveValue.String() != first.Targets["PROTEIN"].ActiveValue {
		t.Fatal("history not preserved", old, e)
	}
}
func TestCustomTargetsPreserveUntouchedFieldsAndRejectStaleVersions(t *testing.T) {
	s, owner, other := phaseTwoServer(t)
	user, _ := Principal(owner)
	s.DB.Collection("user_profiles").UpdateOne(owner, bson.M{"user_id": user}, bson.M{"$set": bson.M{"version": int64(1)}})
	values := map[string]*pb.TargetOverride{}
	for key, value := range map[string]string{"ENERGY": "2000", "PROTEIN": "50.123456789", "CARBOHYDRATE": "275", "FAT": "78", "FIBER": "28"} {
		values[key] = &pb.TargetOverride{ActiveValue: value}
	}
	first, e := s.SaveCustom(owner, &pb.SaveCustomTargetsRequest{ProfileVersion: 1, Changes: values})
	if e != nil {
		t.Fatal(e)
	}
	req := &pb.SaveCustomTargetsRequest{ProfileVersion: 1, ExpectedActiveId: first.Id, ExpectedRevision: first.Revision, Changes: map[string]*pb.TargetOverride{"PROTEIN": {ActiveValue: "150"}}}
	if _, e = s.SaveCustom(other, req); e == nil {
		t.Fatal("foreign save succeeded")
	}
	second, e := s.SaveCustom(owner, req)
	if e != nil {
		t.Fatal(e)
	}
	if second.Targets["ENERGY"].ActiveValue != "2000" || second.Targets["PROTEIN"].ActiveValue != "150" || second.Targets["CARBOHYDRATE"].ActiveValue != first.Targets["CARBOHYDRATE"].ActiveValue || second.InputSnapshot.WeightKg != "70.4375" {
		t.Fatalf("unrelated values changed: %+v", second)
	}
	if _, e = s.SaveCustom(owner, req); status.Code(e) != codes.Aborted {
		t.Fatalf("stale request: %v", e)
	}
	var p profileDoc
	s.DB.Collection("user_profiles").FindOne(owner, bson.M{"user_id": user}).Decode(&p)
	if p.Weight != nil {
		t.Fatal("assumed weight persisted as measurement")
	}
}
func TestSummaryPrecisionAndMeaning(t *testing.T) {
	if got, ok := summarySum([]string{"0.1", "0.2", "0"}); !ok || got != "0.3" {
		t.Fatal(got, ok)
	}
	if _, ok := summarySum(nil); ok {
		t.Fatal("empty became zero")
	}
	if summaryHealthStatus(&pb.HealthOccurrence{Type: "MEDICATION", Status: "MISSED"}) != "overdue — no intake recorded" {
		t.Fatal("overdue implies actual missed dose")
	}
}

func TestSummaryKnownZeroAndPartialFoodCoverage(t *testing.T) {
	s, owner, _ := phaseTwoServer(t)
	user, _ := Principal(owner)
	day := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	group := primitive.NewObjectID()
	zero, _ := primitive.ParseDecimal128("0")
	if _, err := s.DB.Collection("food_log_groups").InsertOne(owner, foodLogGroupDoc{ID: group, UserID: user, Status: "CONFIRMED", TrackingDay: day, TimezoneAtLogging: "Asia/Kolkata", DailyResetTimeAtLog: "04:00", ItemCount: 3, Aggregate: map[string]foodNutrientDoc{"PROTEIN": {Status: "UNKNOWN", Unit: "g"}, "ENERGY": {Status: "KNOWN", Amount: &zero, Unit: "kcal"}}}); err != nil {
		t.Fatal(err)
	}
	for _, amount := range []string{"0.1", "0.2", ""} {
		n := foodNutrientDoc{Status: "UNKNOWN", Unit: "g"}
		if amount != "" {
			value, _ := primitive.ParseDecimal128(amount)
			n.Status = "KNOWN"
			n.Amount = &value
		}
		if _, err := s.DB.Collection("food_log_items").InsertOne(owner, foodLogItemDoc{ID: primitive.NewObjectID(), UserID: user, GroupID: group, TrackingDay: day, FinalNutrition: map[string]foodNutrientDoc{"PROTEIN": n}}); err != nil {
			t.Fatal(err)
		}
	}
	got, err := s.TextSummary(owner, &pb.TextSummaryRequest{FromTrackingDay: day, ToTrackingDay: day, Mode: "DAY"})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"ENERGY: 0 kcal", "Known-food subtotal 0.3 g; coverage 2/3 foods", "PROTEIN: Unknown"} {
		if !strings.Contains(got.Text, want) {
			t.Fatalf("missing %q: %s", want, got.Text)
		}
	}
}
func TestTextSummaryBoundariesPrivacyAndPaging(t *testing.T) {
	s, owner, _ := phaseTwoServer(t)
	user, _ := Principal(owner)
	day := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	for i := 0; i < 55; i++ {
		if _, e := s.DB.Collection("wellness_records").InsertOne(owner, wellnessDoc{ID: primitive.NewObjectID(), UserID: user, Kind: "WATER", Amount: primitive.NewDecimal128(0, 0), Unit: "ml", TrackingDay: day, LoggedAt: time.Now().UTC(), Notes: "SECRET NOTE"}); e != nil {
			t.Fatal(e)
		}
	}
	result, e := s.TextSummary(owner, &pb.TextSummaryRequest{FromTrackingDay: day, ToTrackingDay: day, Mode: "DAY", NutrientFilter: "ENERGY"})
	if e != nil {
		t.Fatal(e)
	}
	if strings.Contains(result.Text, "SECRET NOTE") || !strings.Contains(result.Text, "(55 records)") || !strings.Contains(result.Text, "18:30:00Z") || !strings.Contains(result.Text, "not zero intake") {
		t.Fatal(result.Text)
	}
	if _, e = s.TextSummary(owner, &pb.TextSummaryRequest{FromTrackingDay: "2025-01-01", ToTrackingDay: "2026-01-02", Mode: "RANGE"}); status.Code(e) != codes.InvalidArgument {
		t.Fatalf("unbounded range: %v", e)
	}
}

func TestSummaryFlagsUnavailableSectionWithoutLosingOtherSections(t *testing.T) {
	s, owner, _ := phaseTwoServer(t)
	user, _ := Principal(owner)
	day := time.Now().UTC().Format("2006-01-02")
	_, err := s.DB.Collection("wellness_records").InsertOne(owner, bson.M{"_id": primitive.NewObjectID(), "user_id": user, "tracking_day": day, "kind": "WATER", "amount": bson.M{"corrupt": true}, "logged_at": time.Now().UTC()})
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.TextSummary(owner, &pb.TextSummaryRequest{FromTrackingDay: day, ToTrackingDay: day, Mode: "DAY"})
	if err != nil || !got.Partial || !strings.Contains(got.Text, "Wellness unavailable or incomplete") || !strings.Contains(got.Text, "No recorded nutrition days") {
		t.Fatal(got, err)
	}
}

func TestSummaryFlagsNutritionBeyondTenThousandGroups(t *testing.T) {
	s, owner, _ := phaseTwoServer(t)
	user, _ := Principal(owner)
	day := time.Now().UTC().Format("2006-01-02")
	rows := make([]interface{}, 10001)
	for i := range rows {
		rows[i] = bson.M{"_id": primitive.NewObjectID(), "user_id": user, "status": "CONFIRMED", "tracking_day": day}
	}
	if _, err := s.DB.Collection("food_log_groups").InsertMany(owner, rows); err != nil {
		t.Fatal(err)
	}
	got, err := s.TextSummary(owner, &pb.TextSummaryRequest{FromTrackingDay: day, ToTrackingDay: day, Mode: "DAY"})
	if err != nil || !got.Partial || !strings.Contains(got.Text, "Nutrition unavailable or incomplete") || strings.Contains(got.Text, "No recorded nutrition days") {
		t.Fatal(got, err)
	}
}
