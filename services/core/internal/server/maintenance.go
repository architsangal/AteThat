package server

import (
	"context"
	"math/big"
	"time"

	pb "github.com/architsangal/AteThat/services/core/internal/gen/atethat/v1"
	"github.com/architsangal/AteThat/services/core/internal/targets"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func maintenanceForDay(history []*pb.MaintenanceRevision, day string) *pb.MaintenanceRevision {
	var selected *pb.MaintenanceRevision
	for _, v := range history {
		if v.EffectiveFromTrackingDay <= day && (selected == nil || v.Revision > selected.Revision) {
			selected = v
		}
	}
	return selected
}

func (s *Server) GetMaintenance(ctx context.Context, _ *pb.FoodEmpty) (*pb.MaintenancePlan, error) {
	user, err := Principal(ctx)
	if err != nil {
		return nil, err
	}
	current, err := s.GetCurrentTrackingDay(ctx, &pb.Empty{})
	if err != nil {
		return nil, err
	}
	var settings wellnessSettingsDoc
	err = s.DB.Collection("wellness_settings").FindOne(ctx, bson.M{"user_id": user}).Decode(&settings)
	if err != nil && err != mongo.ErrNoDocuments {
		return nil, internal()
	}
	var profile profileDoc
	if s.DB.Collection("user_profiles").FindOne(ctx, bson.M{"user_id": user}).Decode(&profile) != nil {
		return nil, internal()
	}
	inputs, err := planningProfile(profile, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	inputs.Activity = "SEDENTARY"
	inputs.AssumedActivity = true
	result := &pb.MaintenancePlan{Version: uint64(settings.Version), History: settings.Maintenance, TrackingDay: current.TrackingDay, ProfileVersion: uint64(profile.Version), Inputs: inputs.pb()}
	if profile.DateOfBirth == "" {
		result.UnavailableReason = "Add your date of birth and equation-specific sex to calculate an estimate, or enter maintenance calories manually."
		return result, nil
	}
	result.SuggestedKcal, result.Equation, err = targets.LowActivityMaintenance(inputs.Age, inputs.Sex, inputs.HeightCM, inputs.WeightKG)
	if err != nil {
		result.UnavailableReason = err.Error()
	}
	return result, nil
}

func (s *Server) SaveMaintenance(ctx context.Context, r *pb.MaintenanceSaveRequest) (*pb.MaintenancePlan, error) {
	user, err := Principal(ctx)
	if err != nil {
		return nil, err
	}
	if _, ok := positiveDecimal(r.Kcal); !ok {
		return nil, invalid("Maintenance calories must be a positive number.")
	}
	if r.Mode != "MANUAL" && r.Mode != "CALCULATED" {
		return nil, invalid("Choose a manual or calculated maintenance baseline.")
	}
	plan, err := s.GetMaintenance(ctx, &pb.FoodEmpty{})
	if err != nil {
		return nil, err
	}
	if r.Version != plan.Version {
		return nil, status.Error(codes.Aborted, "Maintenance settings changed. Reload before saving.")
	}
	if len(plan.History) >= 1000 {
		return nil, status.Error(codes.ResourceExhausted, "Maintenance history limit reached")
	}
	if r.Mode == "CALCULATED" && (plan.SuggestedKcal == "" || r.Kcal != plan.SuggestedKcal || r.ProfileVersion != plan.ProfileVersion) {
		return nil, status.Error(codes.Aborted, "Profile estimate changed. Reload and review it before saving.")
	}
	effective := plan.TrackingDay
	if len(plan.History) > 0 {
		day, _ := time.Parse("2006-01-02", effective)
		effective = day.AddDate(0, 0, 1).Format("2006-01-02")
	}
	revision := &pb.MaintenanceRevision{Revision: uint64(len(plan.History) + 1), Kcal: r.Kcal, Mode: r.Mode, EffectiveFromTrackingDay: effective, CreatedAt: time.Now().UTC().Format(time.RFC3339Nano), ProfileVersion: plan.ProfileVersion, HeightCm: plan.Inputs.HeightCm, WeightKg: plan.Inputs.WeightKg, Age: plan.Inputs.Age, Sex: plan.Inputs.Sex}
	if r.Mode == "CALCULATED" {
		revision.Equation = plan.Equation
	}
	collection := s.DB.Collection("wellness_settings")
	if plan.Version == 0 {
		_, err = collection.InsertOne(ctx, wellnessSettingsDoc{UserID: user, Version: 1, Maintenance: []*pb.MaintenanceRevision{revision}})
	} else {
		var result *mongo.UpdateResult
		result, err = collection.UpdateOne(ctx, bson.M{"user_id": user, "version": int64(r.Version)}, bson.M{"$push": bson.M{"maintenance_history": revision}, "$inc": bson.M{"version": 1}})
		if err == nil && result.MatchedCount != 1 {
			return nil, status.Error(codes.Aborted, "Maintenance settings changed. Reload before saving.")
		}
	}
	if mongo.IsDuplicateKeyError(err) {
		return nil, status.Error(codes.Aborted, "Maintenance settings changed. Reload before saving.")
	}
	if err != nil {
		return nil, internal()
	}
	return s.GetMaintenance(ctx, &pb.FoodEmpty{})
}

func deficitValue(maintenance, workout, food string) (string, error) {
	values := []*big.Rat{}
	for _, raw := range []string{maintenance, workout, food} {
		n, ok := new(big.Rat).SetString(raw)
		if !ok || n.Sign() < 0 {
			return "", invalid("invalid calorie amount")
		}
		values = append(values, n)
	}
	value := new(big.Rat).Sub(new(big.Rat).Add(values[0], values[1]), values[2])
	return value.FloatString(6), nil
}

func (s *Server) DailyWellness(ctx context.Context, r *pb.FullAnalysisRequest) (*pb.DailyWellnessPage, error) {
	user, err := Principal(ctx)
	if err != nil {
		return nil, err
	}
	from, to, err := parseDayRange(r.From, r.To)
	if err != nil {
		return nil, err
	}
	if to.Sub(from) > 365*24*time.Hour {
		return nil, invalid("Choose up to 366 inclusive Tracking Days.")
	}
	plan, err := s.GetMaintenance(ctx, &pb.FoodEmpty{})
	if err != nil {
		return nil, err
	}
	settings, err := s.GetSettings(ctx, &pb.FoodEmpty{})
	if err != nil {
		return nil, err
	}
	result := &pb.DailyWellnessPage{Items: []*pb.DailyWellness{}, WaterTargetMl: settings.WaterTargetMl}
	byDay := map[string]*pb.DailyWellness{}
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		day := d.Format("2006-01-02")
		row := &pb.DailyWellness{TrackingDay: day, WorkoutKcal: "0", WorkoutMinutes: "0"}
		if v := maintenanceForDay(plan.History, day); v != nil {
			row.MaintenanceKcal = v.Kcal
			row.MaintenanceRevision = v.Revision
		}
		byDay[day] = row
		result.Items = append(result.Items, row)
	}
	cursor := ""
	for page := 0; page < 8; page++ {
		p, e := s.NutritionHistory(ctx, &pb.NutritionHistoryRequest{FromTrackingDay: r.From, ToTrackingDay: r.To, Limit: 50, Cursor: cursor})
		if e != nil {
			return nil, e
		}
		for _, d := range p.Items {
			if value := d.Totals["ENERGY"]; value != nil && value.Status == "KNOWN" && value.Unit == "kcal" {
				byDay[d.TrackingDay].FoodKcal = value.Amount
			}
		}
		if !p.HasMore {
			break
		}
		if p.NextCursor == "" || p.NextCursor == cursor || page == 7 {
			return nil, internal()
		}
		cursor = p.NextCursor
	}
	rows, err := s.DB.Collection("wellness_records").Find(ctx, bson.M{"user_id": user, "deleted_at": bson.M{"$exists": false}, "tracking_day": bson.M{"$gte": r.From, "$lte": r.To}, "kind": bson.M{"$in": bson.A{"WATER", "ACTIVITY"}}})
	if err != nil {
		return nil, internal()
	}
	defer rows.Close(ctx)
	sum := func(a, b string) (string, error) {
		if a == "" {
			a = "0"
		}
		x, ok := new(big.Rat).SetString(a)
		y, ok2 := new(big.Rat).SetString(b)
		if !ok || !ok2 || x.Sign() < 0 || y.Sign() < 0 {
			return "", internal()
		}
		return new(big.Rat).Add(x, y).FloatString(6), nil
	}
	for rows.Next(ctx) {
		var record wellnessDoc
		if rows.Decode(&record) != nil {
			return nil, internal()
		}
		row := byDay[record.TrackingDay]
		if record.Kind == "WATER" {
			row.WaterMl, err = sum(row.WaterMl, record.Amount.String())
		} else {
			row.WorkoutCount++
			row.WorkoutMinutes, err = sum(row.WorkoutMinutes, record.Amount.String())
			if err != nil {
				return nil, err
			}
			if record.EstimatedKcal == nil {
				row.WorkoutsWithoutCalories++
			} else {
				row.WorkoutKcal, err = sum(row.WorkoutKcal, record.EstimatedKcal.String())
			}
		}
		if err != nil {
			return nil, err
		}
	}
	if rows.Err() != nil {
		return nil, internal()
	}
	for _, row := range result.Items {
		if row.MaintenanceKcal != "" && row.FoodKcal != "" && row.WorkoutsWithoutCalories == 0 {
			row.CalorieDeficit, err = deficitValue(row.MaintenanceKcal, row.WorkoutKcal, row.FoodKcal)
			if err != nil {
				return nil, err
			}
		}
	}
	return result, nil
}
