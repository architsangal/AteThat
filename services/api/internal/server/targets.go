package server

import (
	"context"
	"net/http"
	"strconv"

	pb "github.com/architsangal/AteThat/services/api/internal/gen/atethat/v1"
)

func (s *Server) calculateTargets(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context) {
	user, ok := s.authenticated(r, w, requestID)
	if !ok {
		return
	}
	var body struct {
		ProfileVersion           uint64 `json:"profile_version"`
		EffectiveFromTrackingDay string `json:"effective_from_tracking_day"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		s.problem(w, requestID, 400, "INVALID_REQUEST", "Invalid JSON request")
		return
	}
	result, err := s.Target.Calculate(s.rpcContext(ctx, pb.TargetService_Calculate_FullMethodName, user.UserID, user.SessionFamily),
		&pb.CalculateTargetRequest{ProfileVersion: body.ProfileVersion, EffectiveFromTrackingDay: body.EffectiveFromTrackingDay})
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	s.json(w, 201, targetJSON(result))
}

func targetJSON(value *pb.TargetVersion) interface{} {
	if value == nil {
		return nil
	}
	items := make(map[string]interface{}, len(value.Targets))
	for key, target := range value.Targets {
		items[key] = map[string]interface{}{"reference_value": target.ReferenceValue,
			"active_value": target.ActiveValue, "unit": target.Unit,
			"is_manual_override": target.IsManualOverride, "override_reason": target.OverrideReason, "provenance": target.Provenance, "reference_range": target.ReferenceRange, "outside_reference": target.OutsideReference, "acknowledged_at": target.AcknowledgedAt}
	}
	return map[string]interface{}{
		"id": value.Id, "revision": value.Revision, "status": value.Status,
		"profile_version": value.ProfileVersion,
		"reference_set":   map[string]string{"key": value.ReferenceSetKey, "version": value.ReferenceSetVersion},
		"source_attribution": map[string]string{"name": value.SourceAttribution.Name,
			"publisher": value.SourceAttribution.Publisher, "version": value.SourceAttribution.Version,
			"information_url": value.SourceAttribution.InformationUrl},
		"calculation_version":         value.CalculationVersion,
		"effective_from_tracking_day": value.EffectiveFromTrackingDay,
		"target_basis":                value.TargetBasis, "targets": items,
		"created_at": value.CreatedAt, "reviewed_at": value.ReviewedAt,
		"input_snapshot": value.InputSnapshot,
	}
}

func (s *Server) getTargets(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context) {
	user, ok := s.authenticated(r, w, requestID)
	if !ok {
		return
	}
	result, err := s.Target.Current(s.rpcContext(ctx, pb.TargetService_Current_FullMethodName, user.UserID, user.SessionFamily), &pb.Empty{})
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	s.json(w, 200, map[string]interface{}{"active": targetJSON(result.Active), "proposed": targetJSON(result.Proposed)})
}

func (s *Server) acceptTargets(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context, id string) {
	user, ok := s.authenticated(r, w, requestID)
	if !ok {
		return
	}
	var body struct {
		EffectiveFromTrackingDay string `json:"effective_from_tracking_day"`
		Overrides                map[string]struct {
			ActiveValue    string `json:"active_value"`
			OverrideReason string `json:"override_reason"`
		} `json:"overrides"`
		AdultScopeConfirmed bool `json:"adult_scope_confirmed"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		s.problem(w, requestID, 400, "INVALID_REQUEST", "Invalid JSON request")
		return
	}
	overrides := make(map[string]*pb.TargetOverride, len(body.Overrides))
	for key, value := range body.Overrides {
		overrides[key] = &pb.TargetOverride{ActiveValue: value.ActiveValue, OverrideReason: value.OverrideReason}
	}
	result, err := s.Target.Accept(s.rpcContext(ctx, pb.TargetService_Accept_FullMethodName, user.UserID, user.SessionFamily),
		&pb.AcceptTargetRequest{VersionId: id, EffectiveFromTrackingDay: body.EffectiveFromTrackingDay, Overrides: overrides, AdultScopeConfirmed: body.AdultScopeConfirmed})
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	s.json(w, 200, targetJSON(result))
}

func (s *Server) rejectTargets(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context, id string) {
	user, ok := s.authenticated(r, w, requestID)
	if !ok {
		return
	}
	var body struct{}
	if err := decodeJSON(w, r, &body); err != nil {
		s.problem(w, requestID, 400, "INVALID_REQUEST", "Invalid JSON request")
		return
	}
	_, err := s.Target.Reject(s.rpcContext(ctx, pb.TargetService_Reject_FullMethodName, user.UserID, user.SessionFamily), &pb.TargetVersionRequest{VersionId: id})
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) targetHistory(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context) {
	user, ok := s.authenticated(r, w, requestID)
	if !ok {
		return
	}
	limit := uint32(20)
	if text := r.URL.Query().Get("limit"); text != "" {
		parsed, err := strconv.ParseUint(text, 10, 32)
		if err != nil || parsed == 0 || parsed > 50 {
			s.problem(w, requestID, 422, "VALIDATION_FAILED", "Invalid page limit")
			return
		}
		limit = uint32(parsed)
	}
	result, err := s.Target.History(s.rpcContext(ctx, pb.TargetService_History_FullMethodName, user.UserID, user.SessionFamily),
		&pb.TargetHistoryRequest{Limit: limit, Cursor: r.URL.Query().Get("cursor")})
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	items := make([]interface{}, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, targetJSON(item))
	}
	s.json(w, 200, map[string]interface{}{"items": items, "page": map[string]interface{}{"next_cursor": nilIfEmpty(result.NextCursor), "has_more": result.HasMore}})
}

func nilIfEmpty(value string) interface{} {
	if value == "" {
		return nil
	}
	return value
}
