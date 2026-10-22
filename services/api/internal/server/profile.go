package server

import (
	"context"
	"net/http"

	pb "github.com/architsangal/AteThat/services/api/internal/gen/atethat/v1"
)

type quantityBody struct {
	Amount string `json:"amount"`
	Unit   string `json:"unit"`
}

type profilePatchBody struct {
	OnboardingComplete *bool         `json:"onboarding_complete"`
	Version            uint64        `json:"version"`
	DisplayName        *string       `json:"display_name"`
	DateOfBirth        *string       `json:"date_of_birth"`
	Sex                *string       `json:"sex"`
	Height             *quantityBody `json:"height"`
	Weight             *quantityBody `json:"weight"`
	ActivityLevel      *string       `json:"activity_level"`
	Goal               *string       `json:"goal"`
	Timezone           *string       `json:"timezone"`
}

func (s *Server) authenticated(r *http.Request, w http.ResponseWriter, requestID string) (principal, bool) {
	userID, err := s.principal(r)
	if err != nil {
		s.problem(w, requestID, 401, "UNAUTHENTICATED", "A valid access token is required")
		return principal{}, false
	}
	return userID, true
}

func (s *Server) getProfile(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context) {
	userID, ok := s.authenticated(r, w, requestID)
	if !ok {
		return
	}
	result, err := s.Profile.GetProfile(s.rpcContext(ctx, pb.ProfileService_GetProfile_FullMethodName, userID.UserID, userID.SessionFamily), &pb.Empty{})
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	s.json(w, 200, profileJSON(result))
}

func (s *Server) updateProfile(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context) {
	userID, ok := s.authenticated(r, w, requestID)
	if !ok {
		return
	}
	var body profilePatchBody
	if err := decodeJSON(w, r, &body); err != nil {
		s.problem(w, requestID, 400, "INVALID_REQUEST", "Invalid JSON request")
		return
	}
	req := &pb.UpdateProfileRequest{Version: body.Version, DisplayName: body.DisplayName, DateOfBirth: body.DateOfBirth,
		OnboardingComplete: body.OnboardingComplete, Sex: body.Sex, ActivityLevel: body.ActivityLevel, Goal: body.Goal, Timezone: body.Timezone}
	if body.Height != nil {
		req.Height = &pb.DecimalQuantity{Amount: body.Height.Amount, Unit: body.Height.Unit}
	}
	if body.Weight != nil {
		req.Weight = &pb.DecimalQuantity{Amount: body.Weight.Amount, Unit: body.Weight.Unit}
	}
	result, err := s.Profile.UpdateProfile(s.rpcContext(ctx, pb.ProfileService_UpdateProfile_FullMethodName, userID.UserID, userID.SessionFamily), req)
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	s.json(w, 200, profileJSON(result))
}

func profileJSON(value *pb.Profile) map[string]interface{} {
	result := map[string]interface{}{
		"id": value.Id, "version": value.Version, "display_name": value.DisplayName, "onboarding_complete": value.OnboardingComplete,
		"timezone": value.Timezone, "updated_at": value.UpdatedAt, "has_google_identity": value.HasGoogleIdentity,
	}
	if value.DateOfBirth != "" {
		result["date_of_birth"] = value.DateOfBirth
	}
	if value.Sex != "" {
		result["sex"] = value.Sex
	}
	if value.ActivityLevel != "" {
		result["activity_level"] = value.ActivityLevel
	}
	if value.Goal != "" {
		result["goal"] = value.Goal
	}
	if value.Height != nil {
		result["height"] = map[string]string{"amount": value.Height.Amount, "unit": value.Height.Unit}
	}
	if value.Weight != nil {
		result["weight"] = map[string]string{"amount": value.Weight.Amount, "unit": value.Weight.Unit}
	}
	return result
}

type preferencesPatchBody struct {
	Version               uint64    `json:"version"`
	DailyResetTime        *string   `json:"daily_reset_time"`
	UnitSystem            *string   `json:"unit_system"`
	DashboardNutrients    *[]string `json:"dashboard_nutrients"`
	AnalysisHealthItemIDs *[]string `json:"analysis_health_item_ids"`
}

func (s *Server) getPreferences(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context) {
	userID, ok := s.authenticated(r, w, requestID)
	if !ok {
		return
	}
	result, err := s.Profile.GetPreferences(s.rpcContext(ctx, pb.ProfileService_GetPreferences_FullMethodName, userID.UserID, userID.SessionFamily), &pb.Empty{})
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	s.json(w, 200, preferencesJSON(result))
}

func (s *Server) updatePreferences(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context) {
	userID, ok := s.authenticated(r, w, requestID)
	if !ok {
		return
	}
	var body preferencesPatchBody
	if err := decodeJSON(w, r, &body); err != nil {
		s.problem(w, requestID, 400, "INVALID_REQUEST", "Invalid JSON request")
		return
	}
	req := &pb.UpdatePreferencesRequest{Version: body.Version, DailyResetTime: body.DailyResetTime, UnitSystem: body.UnitSystem}
	if body.DashboardNutrients != nil {
		req.DashboardNutrients = &pb.NutrientKeys{Values: *body.DashboardNutrients}
	}
	if body.AnalysisHealthItemIDs != nil {
		req.AnalysisHealthItems = &pb.HealthItemSelection{Ids: *body.AnalysisHealthItemIDs}
	}
	result, err := s.Profile.UpdatePreferences(s.rpcContext(ctx, pb.ProfileService_UpdatePreferences_FullMethodName, userID.UserID, userID.SessionFamily), req)
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	s.json(w, 200, preferencesJSON(result))
}

func preferencesJSON(value *pb.Preferences) map[string]interface{} {
	return map[string]interface{}{"id": value.Id, "version": value.Version, "daily_reset_time": value.DailyResetTime,
		"unit_system": value.UnitSystem, "dashboard_nutrients": value.DashboardNutrients,
		"analysis_health_item_ids": append([]string{}, value.AnalysisHealthItemIds...)}
}

func (s *Server) getCurrentTrackingDay(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context) {
	userID, ok := s.authenticated(r, w, requestID)
	if !ok {
		return
	}
	result, err := s.Profile.GetCurrentTrackingDay(s.rpcContext(ctx, pb.ProfileService_GetCurrentTrackingDay_FullMethodName, userID.UserID, userID.SessionFamily), &pb.Empty{})
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	s.json(w, 200, map[string]interface{}{
		"tracking_day": result.TrackingDay, "timezone": result.Timezone,
		"daily_reset_time": result.DailyResetTime, "rule_version": result.RuleVersion,
		"as_of_utc": result.AsOfUtc,
	})
}
