package server

import (
	"context"
	"regexp"
	"strings"
	"time"

	pb "github.com/architsangal/AteThat/services/core/internal/gen/atethat/v1"
	"github.com/architsangal/AteThat/services/core/internal/trackingday"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var localTimePattern = regexp.MustCompile(`^([01][0-9]|2[0-3]):[0-5][0-9]$`)
var nonnegativeDecimalPattern = regexp.MustCompile(`^[0-9]+(\.[0-9]+)?$`)

func profileResponse(doc profileDoc) *pb.Profile {
	result := &pb.Profile{Id: doc.ID.Hex(), Version: uint64(doc.Version), DisplayName: doc.DisplayName,
		DateOfBirth: doc.DateOfBirth, Sex: doc.Sex, ActivityLevel: doc.ActivityLevel, Goal: doc.Goal,
		Timezone: doc.Timezone, UpdatedAt: doc.UpdatedAt.UTC().Format(time.RFC3339Nano)}
	if doc.Height != nil {
		result.Height = &pb.DecimalQuantity{Amount: doc.Height.Amount.String(), Unit: doc.Height.Unit}
	}
	if doc.Weight != nil {
		result.Weight = &pb.DecimalQuantity{Amount: doc.Weight.Amount.String(), Unit: doc.Weight.Unit}
	}
	return result
}

func preferencesResponse(doc preferencesDoc) *pb.Preferences {
	return &pb.Preferences{Id: doc.ID.Hex(), Version: uint64(doc.Version), DailyResetTime: doc.DailyResetTime,
		UnitSystem: doc.UnitSystem, DashboardNutrients: doc.DashboardNutrients}
}

func (s *Server) GetProfile(ctx context.Context, _ *pb.Empty) (*pb.Profile, error) {
	userID, err := Principal(ctx)
	if err != nil {
		return nil, err
	}
	var doc profileDoc
	if err := s.DB.Collection("user_profiles").FindOne(ctx, bson.M{"user_id": userID}).Decode(&doc); err != nil {
		return nil, status.Error(codes.NotFound, "profile not found")
	}
	return profileResponse(doc), nil
}

func (s *Server) UpdateProfile(ctx context.Context, req *pb.UpdateProfileRequest) (*pb.Profile, error) {
	userID, err := Principal(ctx)
	if err != nil {
		return nil, err
	}
	if req.Version == 0 {
		return nil, invalid("version is required")
	}
	set := bson.M{"updated_at": time.Now().UTC()}
	if req.DisplayName != nil {
		value := strings.TrimSpace(req.GetDisplayName())
		if len(value) < 1 || len(value) > 80 {
			return nil, invalid("invalid display name")
		}
		set["display_name"] = value
	}
	if req.DateOfBirth != nil {
		value := req.GetDateOfBirth()
		if value != "" {
			if _, err := time.Parse("2006-01-02", value); err != nil {
				return nil, invalid("invalid date of birth")
			}
		}
		set["date_of_birth"] = value
	}
	if req.Sex != nil {
		set["sex"] = req.GetSex()
	}
	if req.ActivityLevel != nil {
		set["activity_level"] = req.GetActivityLevel()
	}
	if req.Goal != nil {
		set["goal"] = req.GetGoal()
	}
	if req.Timezone != nil {
		if !validTimezone(req.GetTimezone()) {
			return nil, invalid("invalid timezone")
		}
		set["timezone"] = req.GetTimezone()
	}
	if req.Height != nil {
		q, err := parseQuantity(req.Height)
		if err != nil {
			return nil, err
		}
		set["height"] = q
	}
	if req.Weight != nil {
		q, err := parseQuantity(req.Weight)
		if err != nil {
			return nil, err
		}
		set["weight"] = q
	}
	var doc profileDoc
	filter := bson.M{"user_id": userID, "version": int64(req.Version)}
	err = s.DB.Collection("user_profiles").FindOneAndUpdate(ctx, filter, bson.M{"$set": set, "$inc": bson.M{"version": 1}},
		options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&doc)
	if err == mongo.ErrNoDocuments {
		return nil, status.Error(codes.Aborted, "profile version conflict")
	}
	if err != nil {
		return nil, internal()
	}
	return profileResponse(doc), nil
}

func parseQuantity(q *pb.DecimalQuantity) (*quantityDoc, error) {
	if q.Unit == "" || !nonnegativeDecimalPattern.MatchString(q.Amount) {
		return nil, invalid("quantity unit is required")
	}
	decimal, err := primitive.ParseDecimal128(q.Amount)
	if err != nil {
		return nil, invalid("invalid quantity amount")
	}
	return &quantityDoc{Amount: decimal, Unit: q.Unit}, nil
}

func (s *Server) GetPreferences(ctx context.Context, _ *pb.Empty) (*pb.Preferences, error) {
	userID, err := Principal(ctx)
	if err != nil {
		return nil, err
	}
	var doc preferencesDoc
	if err := s.DB.Collection("user_preferences").FindOne(ctx, bson.M{"user_id": userID}).Decode(&doc); err != nil {
		return nil, status.Error(codes.NotFound, "preferences not found")
	}
	return preferencesResponse(doc), nil
}

func (s *Server) GetCurrentTrackingDay(ctx context.Context, _ *pb.Empty) (*pb.TrackingDayContext, error) {
	userID, err := Principal(ctx)
	if err != nil {
		return nil, err
	}
	var profile profileDoc
	if err := s.DB.Collection("user_profiles").FindOne(ctx, bson.M{"user_id": userID}).Decode(&profile); err != nil {
		return nil, status.Error(codes.NotFound, "profile not found")
	}
	var preferences preferencesDoc
	if err := s.DB.Collection("user_preferences").FindOne(ctx, bson.M{"user_id": userID}).Decode(&preferences); err != nil {
		return nil, status.Error(codes.NotFound, "preferences not found")
	}
	result, err := trackingday.Derive(time.Now().UTC(), profile.Timezone, preferences.DailyResetTime)
	if err != nil {
		return nil, internal()
	}
	return &pb.TrackingDayContext{TrackingDay: result.Day, Timezone: result.Timezone,
		DailyResetTime: result.DailyResetTime, RuleVersion: result.RuleVersion,
		AsOfUtc: result.AsOfUTC.Format(time.RFC3339Nano)}, nil
}

func (s *Server) UpdatePreferences(ctx context.Context, req *pb.UpdatePreferencesRequest) (*pb.Preferences, error) {
	userID, err := Principal(ctx)
	if err != nil {
		return nil, err
	}
	if req.Version == 0 {
		return nil, invalid("version is required")
	}
	set := bson.M{"updated_at": time.Now().UTC()}
	if req.DailyResetTime != nil {
		if !localTimePattern.MatchString(req.GetDailyResetTime()) {
			return nil, invalid("invalid daily reset time")
		}
		set["daily_reset_time"] = req.GetDailyResetTime()
	}
	if req.UnitSystem != nil {
		if req.GetUnitSystem() != "METRIC" && req.GetUnitSystem() != "IMPERIAL" {
			return nil, invalid("invalid unit system")
		}
		set["unit_system"] = req.GetUnitSystem()
	}
	if req.DashboardNutrients != nil {
		if len(req.DashboardNutrients.Values) > 20 {
			return nil, invalid("too many dashboard nutrients")
		}
		set["dashboard_nutrients"] = req.DashboardNutrients.Values
	}
	var doc preferencesDoc
	err = s.DB.Collection("user_preferences").FindOneAndUpdate(ctx,
		bson.M{"user_id": userID, "version": int64(req.Version)}, bson.M{"$set": set, "$inc": bson.M{"version": 1}},
		options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&doc)
	if err == mongo.ErrNoDocuments {
		return nil, status.Error(codes.Aborted, "preferences version conflict")
	}
	if err != nil {
		return nil, internal()
	}
	return preferencesResponse(doc), nil
}
