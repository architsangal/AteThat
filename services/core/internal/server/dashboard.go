package server

import (
	"context"
	"time"

	"github.com/architsangal/AteThat/services/core/internal/foods"
	pb "github.com/architsangal/AteThat/services/core/internal/gen/atethat/v1"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (s *Server) GetDashboard(ctx context.Context, req *pb.DashboardRequest) (*pb.DashboardResponse, error) {
	userID, err := Principal(ctx)
	if err != nil {
		return nil, err
	}
	day := req.TrackingDay
	if day == "" {
		current, err := s.GetCurrentTrackingDay(ctx, &pb.Empty{})
		if err != nil {
			return nil, err
		}
		day = current.TrackingDay
	}
	if parsed, err := time.Parse("2006-01-02", day); err != nil || parsed.Format("2006-01-02") != day {
		return nil, invalid("invalid tracking day")
	}
	var profile profileDoc
	var preferences preferencesDoc
	if s.DB.Collection("user_profiles").FindOne(ctx, bson.M{"user_id": userID}).Decode(&profile) != nil ||
		s.DB.Collection("user_preferences").FindOne(ctx, bson.M{"user_id": userID}).Decode(&preferences) != nil {
		return nil, internal()
	}
	result := &pb.DashboardResponse{TrackingDay: day, Timezone: profile.Timezone,
		DailyResetTime: preferences.DailyResetTime, Totals: map[string]*pb.FoodNutrient{},
		Targets: map[string]*pb.FoodNutrient{}, Progress: map[string]*pb.NutritionProgress{},
		FoodLogGroups: []*pb.FoodLogGroup{}}
	rows, err := s.DB.Collection("food_log_groups").Find(ctx,
		bson.M{"user_id": userID, "tracking_day": day, "status": "CONFIRMED"},
		options.Find().SetSort(bson.D{{Key: "logged_at_utc", Value: -1}, {Key: "_id", Value: -1}}))
	if err != nil {
		return nil, internal()
	}
	defer rows.Close(ctx)
	groups := []foodLogItemDoc{}
	for rows.Next(ctx) {
		var group foodLogGroupDoc
		if rows.Decode(&group) != nil {
			return nil, internal()
		}
		groups = append(groups, foodLogItemDoc{FinalNutrition: group.Aggregate})
		if len(result.FoodLogGroups) < 5 {
			result.FoodLogGroups = append(result.FoodLogGroups, groupPB(group, nil))
		} else {
			result.FoodGroupsHasMore = true
		}
	}
	if rows.Err() != nil {
		return nil, internal()
	}
	if len(groups) > 0 {
		totals, err := aggregateNutrition(groups)
		if err != nil {
			return nil, internal()
		}
		result.Totals = nutrientPB(totals)
	} else {
		var summary dailyNutritionSummaryDoc
		err := s.DB.Collection("daily_nutrition_summaries").FindOne(ctx,
			bson.M{"user_id": userID, "tracking_day": day}).Decode(&summary)
		if err != nil && err != mongo.ErrNoDocuments {
			return nil, internal()
		}
		if err == nil {
			result.Totals = nutrientPB(summary.Aggregate)
		}
	}
	var target targetVersionDoc
	err = s.DB.Collection("nutrition_target_versions").FindOne(ctx,
		bson.M{"user_id": userID, "status": "ACTIVE", "effective_from_tracking_day": bson.M{"$lte": day}}).Decode(&target)
	if err != nil && err != mongo.ErrNoDocuments {
		return nil, internal()
	}
	if err == nil {
		for key, value := range target.Targets {
			result.Targets[key] = &pb.FoodNutrient{Status: "KNOWN", Amount: value.ActiveValue.String(), Unit: value.Unit}
			consumed := result.Totals[key]
			if consumed == nil || consumed.Status != "KNOWN" || consumed.Unit != value.Unit {
				continue
			}
			percent, err := foods.Scale(consumed.Amount, "100", value.ActiveValue.String())
			if err != nil {
				continue
			}
			result.Progress[key] = &pb.NutritionProgress{ConsumedAmount: consumed.Amount,
				TargetAmount: value.ActiveValue.String(), Unit: value.Unit, Percent: percent}
		}
	}
	return result, nil
}
