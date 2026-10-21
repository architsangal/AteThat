package server

import (
	"context"
	pb "github.com/architsangal/AteThat/services/api/internal/gen/atethat/v1"
	"net/http"
	"strings"
)

func (s *Server) targetPlanning(w http.ResponseWriter, r *http.Request, id string, ctx context.Context) {
	user, ok := s.authenticated(r, w, id)
	if !ok {
		return
	}
	var body struct {
		Macros map[string]string `json:"macros"`
	}
	if r.Method == http.MethodPost && decodeJSON(w, r, &body) != nil {
		s.problem(w, id, 400, "INVALID_REQUEST", "Invalid JSON request")
		return
	}
	p, err := s.Target.Planning(s.rpcContext(ctx, pb.TargetService_Planning_FullMethodName, user.UserID, user.SessionFamily), &pb.TargetPlanningRequest{Macros: body.Macros})
	if err != nil {
		s.rpcProblem(w, id, err)
		return
	}
	s.json(w, 200, map[string]interface{}{"inputs": p.Inputs, "profile_version": p.ProfileVersion, "policy_version": p.PolicyVersion, "personalized_enabled": p.PersonalizedEnabled, "pending_decisions": p.PendingDecisions, "missing_inputs": p.MissingInputs, "macro_calories": p.MacroCalories, "rounding_tolerance_kcal": p.RoundingToleranceKcal, "explanation": p.Explanation, "calculated_defaults": p.CalculatedDefaults, "energy_equation": p.EnergyEquation, "protein_rate": p.ProteinRate, "activity_category": p.ActivityCategory, "reset_preview_token": p.ResetPreviewToken, "calculation_unavailable_reason": p.CalculationUnavailableReason, "calculated_warnings": p.CalculatedWarnings})
}
func (s *Server) resetCalculatedTargets(w http.ResponseWriter, r *http.Request, id string, ctx context.Context) {
	user, ok := s.authenticated(r, w, id)
	if !ok {
		return
	}
	var body struct {
		ExpectedActiveID    string `json:"expected_active_id"`
		ExpectedRevision    uint64 `json:"expected_revision"`
		ProfileVersion      uint64 `json:"profile_version"`
		PreviewToken        string `json:"preview_token"`
		AdultScopeConfirmed bool   `json:"adult_scope_confirmed"`
	}
	if decodeJSON(w, r, &body) != nil {
		s.problem(w, id, 400, "INVALID_REQUEST", "Invalid JSON request")
		return
	}
	result, err := s.Target.ResetCalculated(s.rpcContext(ctx, pb.TargetService_ResetCalculated_FullMethodName, user.UserID, user.SessionFamily), &pb.ResetCalculatedTargetsRequest{ExpectedActiveId: body.ExpectedActiveID, ExpectedRevision: body.ExpectedRevision, ProfileVersion: body.ProfileVersion, PreviewToken: body.PreviewToken, AdultScopeConfirmed: body.AdultScopeConfirmed})
	if err != nil {
		s.rpcProblem(w, id, err)
		return
	}
	s.json(w, 201, targetJSON(result))
}
func (s *Server) saveCustomTargets(w http.ResponseWriter, r *http.Request, id string, ctx context.Context) {
	user, ok := s.authenticated(r, w, id)
	if !ok {
		return
	}
	var body struct {
		ExpectedActiveID string `json:"expected_active_id"`
		ExpectedRevision uint64 `json:"expected_revision"`
		ProfileVersion   uint64 `json:"profile_version"`
		Changes          map[string]struct {
			ActiveValue    string `json:"active_value"`
			OverrideReason string `json:"override_reason"`
		} `json:"changes"`
		EnergyProvenance     string `json:"energy_provenance"`
		AcknowledgementToken string `json:"acknowledgement_token"`
	}
	if decodeJSON(w, r, &body) != nil {
		s.problem(w, id, 400, "INVALID_REQUEST", "Invalid JSON request")
		return
	}
	changes := map[string]*pb.TargetOverride{}
	for k, v := range body.Changes {
		changes[k] = &pb.TargetOverride{ActiveValue: v.ActiveValue, OverrideReason: v.OverrideReason}
	}
	request := &pb.SaveCustomTargetsRequest{ExpectedActiveId: body.ExpectedActiveID, ExpectedRevision: body.ExpectedRevision, ProfileVersion: body.ProfileVersion, Changes: changes, EnergyProvenance: body.EnergyProvenance, AcknowledgementToken: body.AcknowledgementToken}
	if strings.HasSuffix(r.URL.Path, "/review") {
		result, err := s.Target.ReviewCustom(s.rpcContext(ctx, pb.TargetService_ReviewCustom_FullMethodName, user.UserID, user.SessionFamily), request)
		if err != nil {
			s.rpcProblem(w, id, err)
			return
		}
		s.json(w, 200, map[string]interface{}{"warnings": result.Warnings, "acknowledgement_token": result.AcknowledgementToken})
		return
	}
	result, err := s.Target.SaveCustom(s.rpcContext(ctx, pb.TargetService_SaveCustom_FullMethodName, user.UserID, user.SessionFamily), request)
	if err != nil {
		s.rpcProblem(w, id, err)
		return
	}
	s.json(w, 201, targetJSON(result))
}
func (s *Server) textSummary(w http.ResponseWriter, r *http.Request, id string, ctx context.Context) {
	user, ok := s.authenticated(r, w, id)
	if !ok {
		return
	}
	q := r.URL.Query()
	p, err := s.Analysis.TextSummary(s.rpcContext(ctx, pb.AnalysisService_TextSummary_FullMethodName, user.UserID, user.SessionFamily), &pb.TextSummaryRequest{FromTrackingDay: q.Get("from"), ToTrackingDay: q.Get("to"), Mode: q.Get("mode"), NutrientFilter: q.Get("nutrient")})
	if err != nil {
		s.rpcProblem(w, id, err)
		return
	}
	s.json(w, 200, map[string]interface{}{"text": p.Text, "generated_at": p.GeneratedAt, "partial": p.Partial, "context": p.Context})
}

func (s *Server) proposeCalculatedTargets(w http.ResponseWriter, r *http.Request, id string, ctx context.Context) {
	user, ok := s.authenticated(r, w, id)
	if !ok {
		return
	}
	var body struct{}
	if decodeJSON(w, r, &body) != nil {
		s.problem(w, id, 400, "INVALID_REQUEST", "Invalid JSON request")
		return
	}
	result, err := s.Target.ProposeCalculated(s.rpcContext(ctx, pb.TargetService_ProposeCalculated_FullMethodName, user.UserID, user.SessionFamily), &pb.Empty{})
	if err != nil {
		s.rpcProblem(w, id, err)
		return
	}
	s.json(w, 200, targetJSON(result))
}
