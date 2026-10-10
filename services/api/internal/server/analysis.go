package server

import (
	"context"
	"net/http"
	"strconv"

	pb "github.com/architsangal/AteThat/services/api/internal/gen/atethat/v1"
)

func analysisLimit(w http.ResponseWriter, s *Server, requestID, raw string, fallback, maximum uint32) (uint32, bool) {
	if raw == "" {
		return fallback, true
	}
	n, err := strconv.ParseUint(raw, 10, 32)
	if err != nil || n == 0 || n > uint64(maximum) {
		s.problem(w, requestID, 422, "VALIDATION_FAILED", "Invalid page limit")
		return 0, false
	}
	return uint32(n), true
}

func (s *Server) nutritionHistory(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context) {
	user, ok := s.authenticated(r, w, requestID)
	if !ok {
		return
	}
	limit, ok := analysisLimit(w, s, requestID, r.URL.Query().Get("limit"), 30, 50)
	if !ok {
		return
	}
	q := r.URL.Query()
	page, err := s.Analysis.NutritionHistory(s.rpcContext(ctx, pb.AnalysisService_NutritionHistory_FullMethodName, user.UserID, user.SessionFamily),
		&pb.NutritionHistoryRequest{FromTrackingDay: q.Get("from_tracking_day"), ToTrackingDay: q.Get("to_tracking_day"), Limit: limit, Cursor: q.Get("cursor")})
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	items := make([]interface{}, 0, len(page.Items))
	for _, day := range page.Items {
		items = append(items, map[string]interface{}{"tracking_day": day.TrackingDay, "timezone": day.Timezone,
			"daily_reset_time": day.DailyResetTime, "totals": logNutritionJSON(day.Totals),
			"confirmed_group_count": day.ConfirmedGroupCount, "confirmed_item_count": day.ConfirmedItemCount,
			"detail_compacted": day.DetailCompacted})
	}
	s.json(w, 200, map[string]interface{}{"items": items, "page": map[string]interface{}{"next_cursor": nilIfEmpty(page.NextCursor), "has_more": page.HasMore}})
}

func (s *Server) fullAnalysis(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context) {
	user, ok := s.authenticated(r, w, requestID)
	if !ok {
		return
	}
	q := r.URL.Query()
	result, err := s.Analysis.FullAnalysis(s.rpcContext(ctx, pb.AnalysisService_FullAnalysis_FullMethodName, user.UserID, user.SessionFamily),
		&pb.FullAnalysisRequest{From: q.Get("from"), To: q.Get("to")})
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	s.json(w, 200, map[string]interface{}{"from": result.From, "to": result.To, "totals": logNutritionJSON(result.Totals),
		"confirmed_group_count": result.ConfirmedGroupCount, "confirmed_item_count": result.ConfirmedItemCount,
		"compacted_day_count": result.CompactedDayCount})
}

func (s *Server) healthActionHistory(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context) {
	user, ok := s.authenticated(r, w, requestID)
	if !ok {
		return
	}
	limit, ok := analysisLimit(w, s, requestID, r.URL.Query().Get("limit"), 50, 100)
	if !ok {
		return
	}
	q := r.URL.Query()
	page, err := s.Analysis.HealthActionHistory(s.rpcContext(ctx, pb.AnalysisService_HealthActionHistory_FullMethodName, user.UserID, user.SessionFamily),
		&pb.HealthHistoryRequest{From: q.Get("from"), To: q.Get("to"), Limit: limit, Cursor: q.Get("cursor")})
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	items := make([]interface{}, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, healthOccurrenceJSON(item))
	}
	s.json(w, 200, map[string]interface{}{"items": items, "page": map[string]interface{}{"next_cursor": nilIfEmpty(page.NextCursor), "has_more": page.HasMore}})
}
