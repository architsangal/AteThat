package server

import (
	"context"
	"net/http"
	"strconv"

	pb "github.com/architsangal/AteThat/services/api/internal/gen/atethat/v1"
)

type healthItemBody struct {
	Version uint64 `json:"version"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	Dosage  struct {
		AmountText string `json:"amount_text"`
		UnitText   string `json:"unit_text"`
	} `json:"dosage"`
	Instructions string `json:"instructions"`
	Notes        string `json:"notes"`
	StartDate    string `json:"start_date"`
	Schedule     *struct {
		Timezone   string `json:"timezone"`
		Recurrence struct {
			Frequency  string   `json:"frequency"`
			Interval   uint32   `json:"interval"`
			Weekdays   []string `json:"weekdays"`
			LocalTimes []string `json:"local_times"`
		} `json:"recurrence"`
		EffectiveFromDate string `json:"effective_from_date"`
	} `json:"schedule"`
}

func healthItemRequest(id string, body healthItemBody) *pb.UpsertHealthItemRequest {
	req := &pb.UpsertHealthItemRequest{Id: id, Version: body.Version, Type: body.Type, Name: body.Name,
		DosageAmountText: body.Dosage.AmountText, DosageUnitText: body.Dosage.UnitText,
		Instructions: body.Instructions, Notes: body.Notes, StartDate: body.StartDate}
	if body.Schedule != nil {
		req.Schedule = &pb.HealthSchedule{Timezone: body.Schedule.Timezone,
			Frequency: body.Schedule.Recurrence.Frequency, Interval: body.Schedule.Recurrence.Interval,
			Weekdays: body.Schedule.Recurrence.Weekdays, LocalTimes: body.Schedule.Recurrence.LocalTimes,
			EffectiveFromDate: body.Schedule.EffectiveFromDate}
	}
	return req
}

func healthItemJSON(item *pb.HealthItem) interface{} {
	return map[string]interface{}{"id": item.Id, "version": item.Version, "type": item.Type, "name": item.Name,
		"dosage":       map[string]string{"amount_text": item.DosageAmountText, "unit_text": item.DosageUnitText},
		"instructions": item.Instructions, "notes": item.Notes, "active": item.Active, "start_date": item.StartDate,
		"end_date": nilIfEmpty(item.EndDate), "schedule": map[string]interface{}{"timezone": item.Schedule.Timezone,
			"recurrence": map[string]interface{}{"frequency": item.Schedule.Frequency, "interval": item.Schedule.Interval,
				"weekdays": item.Schedule.Weekdays, "local_times": item.Schedule.LocalTimes},
			"effective_from_date": item.Schedule.EffectiveFromDate, "version": item.Schedule.Version},
		"created_at": item.CreatedAt, "updated_at": item.UpdatedAt}
}

func (s *Server) createHealthItem(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context) {
	user, ok := s.authenticated(r, w, requestID)
	if !ok {
		return
	}
	var body healthItemBody
	if decodeJSON(w, r, &body) != nil {
		s.problem(w, requestID, 400, "INVALID_REQUEST", "Invalid JSON request")
		return
	}
	item, err := s.Health.CreateItem(s.rpcContext(ctx, pb.HealthService_CreateItem_FullMethodName, user.UserID, user.SessionFamily), healthItemRequest("", body))
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	w.Header().Set("Location", "/api/v1/health-items/"+item.Id)
	s.json(w, 201, healthItemJSON(item))
}

func (s *Server) updateHealthItem(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context, id string) {
	user, ok := s.authenticated(r, w, requestID)
	if !ok {
		return
	}
	var body healthItemBody
	if decodeJSON(w, r, &body) != nil {
		s.problem(w, requestID, 400, "INVALID_REQUEST", "Invalid JSON request")
		return
	}
	item, err := s.Health.UpdateItem(s.rpcContext(ctx, pb.HealthService_UpdateItem_FullMethodName, user.UserID, user.SessionFamily), healthItemRequest(id, body))
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	s.json(w, 200, healthItemJSON(item))
}

func (s *Server) getHealthItem(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context, id string) {
	user, ok := s.authenticated(r, w, requestID)
	if !ok {
		return
	}
	item, err := s.Health.GetItem(s.rpcContext(ctx, pb.HealthService_GetItem_FullMethodName, user.UserID, user.SessionFamily), &pb.HealthItemRequest{Id: id})
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	s.json(w, 200, healthItemJSON(item))
}

func (s *Server) listHealthItems(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context) {
	user, ok := s.authenticated(r, w, requestID)
	if !ok {
		return
	}
	limit := uint32(20)
	if raw := r.URL.Query().Get("limit"); raw != "" {
		value, err := strconv.ParseUint(raw, 10, 32)
		if err != nil || value < 1 || value > 50 {
			s.problem(w, requestID, 422, "VALIDATION_FAILED", "Invalid page limit")
			return
		}
		limit = uint32(value)
	}
	page, err := s.Health.ListItems(s.rpcContext(ctx, pb.HealthService_ListItems_FullMethodName, user.UserID, user.SessionFamily),
		&pb.ListHealthItemsRequest{Limit: limit, Cursor: r.URL.Query().Get("cursor"), ActiveFilter: r.URL.Query().Get("active")})
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	items := make([]interface{}, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, healthItemJSON(item))
	}
	s.json(w, 200, map[string]interface{}{"items": items, "page": map[string]interface{}{"next_cursor": nilIfEmpty(page.NextCursor), "has_more": page.HasMore}})
}

func (s *Server) deactivateHealthItem(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context, id string) {
	user, ok := s.authenticated(r, w, requestID)
	if !ok {
		return
	}
	var body struct {
		Version uint64 `json:"version"`
		EndDate string `json:"end_date"`
	}
	if decodeJSON(w, r, &body) != nil {
		s.problem(w, requestID, 400, "INVALID_REQUEST", "Invalid JSON request")
		return
	}
	item, err := s.Health.DeactivateItem(s.rpcContext(ctx, pb.HealthService_DeactivateItem_FullMethodName, user.UserID, user.SessionFamily),
		&pb.DeactivateHealthItemRequest{Id: id, Version: body.Version, EndDate: body.EndDate})
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	s.json(w, 200, healthItemJSON(item))
}

func healthOccurrenceJSON(item *pb.HealthOccurrence) interface{} {
	return map[string]interface{}{"occurrence_id": item.OccurrenceId, "health_item_id": item.HealthItemId,
		"schedule_version": item.ScheduleVersion, "name": item.Name, "type": item.Type, "scheduled_at": item.ScheduledAt,
		"scheduled_local": map[string]string{"date": item.ScheduledLocalDate, "time": item.ScheduledLocalTime, "timezone": item.Timezone},
		"status":          item.Status, "completed_at": nilIfEmpty(item.CompletedAt), "rescheduled_to": nilIfEmpty(item.RescheduledTo), "notes": item.Notes}
}

func (s *Server) listHealthOccurrences(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context) {
	user, ok := s.authenticated(r, w, requestID)
	if !ok {
		return
	}
	limit := uint32(100)
	if raw := r.URL.Query().Get("limit"); raw != "" {
		value, err := strconv.ParseUint(raw, 10, 32)
		if err != nil || value < 1 || value > 100 {
			s.problem(w, requestID, 422, "VALIDATION_FAILED", "Invalid page limit")
			return
		}
		limit = uint32(value)
	}
	page, err := s.Health.ListOccurrences(s.rpcContext(ctx, pb.HealthService_ListOccurrences_FullMethodName, user.UserID, user.SessionFamily),
		&pb.ListOccurrencesRequest{From: r.URL.Query().Get("from"), To: r.URL.Query().Get("to"), Limit: limit, Cursor: r.URL.Query().Get("cursor")})
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

func (s *Server) recordHealthOccurrence(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context, id string) {
	user, ok := s.authenticated(r, w, requestID)
	if !ok {
		return
	}
	var body struct {
		Status        string `json:"status"`
		CompletedAt   string `json:"completed_at"`
		RescheduledTo string `json:"rescheduled_to"`
		Notes         string `json:"notes"`
	}
	if decodeJSON(w, r, &body) != nil {
		s.problem(w, requestID, 400, "INVALID_REQUEST", "Invalid JSON request")
		return
	}
	item, err := s.Health.RecordOccurrence(s.rpcContext(ctx, pb.HealthService_RecordOccurrence_FullMethodName, user.UserID, user.SessionFamily),
		&pb.RecordOccurrenceRequest{OccurrenceId: id, Status: body.Status, CompletedAt: body.CompletedAt, RescheduledTo: body.RescheduledTo, Notes: body.Notes})
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	s.json(w, 200, healthOccurrenceJSON(item))
}
