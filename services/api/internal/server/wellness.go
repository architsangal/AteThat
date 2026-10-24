package server

import (
	"context"
	pb "github.com/architsangal/AteThat/services/api/internal/gen/atethat/v1"
	"net/http"
	"strconv"
	"strings"
)

type wellnessBody struct {
	Version       uint64 `json:"version"`
	Kind          string `json:"kind"`
	Amount        string `json:"amount"`
	Unit          string `json:"unit"`
	LoggedAt      string `json:"logged_at"`
	Name          string `json:"name"`
	Intensity     string `json:"intensity"`
	EstimatedKcal string `json:"estimated_kcal"`
	Notes         string `json:"notes"`
}

func wellnessJSON(r *pb.WellnessRecord) interface{} {
	return map[string]interface{}{"id": r.Id, "version": r.Version, "kind": r.Kind, "amount": r.Amount, "unit": r.Unit, "logged_at": r.LoggedAt, "tracking_day": r.TrackingDay, "timezone": r.Timezone, "daily_reset_time": r.DailyResetTime, "name": r.Name, "intensity": r.Intensity, "estimated_kcal": r.EstimatedKcal, "notes": r.Notes}
}
func (s *Server) wellnessRoute(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context, path string) {
	user, ok := s.authenticated(r, w, requestID)
	if !ok {
		return
	}
	rpc := func(method string) context.Context { return s.rpcContext(ctx, method, user.UserID, user.SessionFamily) }
	if path == "/wellness/maintenance" || path == "/wellness/analysis" {
		s.wellnessAnalysisRoute(w, r, requestID, rpc, path)
		return
	}
	if path == "/wellness/settings" {
		var result *pb.WellnessSettings
		var err error
		if r.Method == http.MethodGet {
			result, err = s.Wellness.GetSettings(rpc(pb.WellnessService_GetSettings_FullMethodName), &pb.FoodEmpty{})
		} else if r.Method == http.MethodPut {
			var b struct {
				WaterTarget string `json:"water_target_ml"`
				Version     uint64 `json:"version"`
			}
			if decodeJSON(w, r, &b) != nil {
				s.problem(w, requestID, 400, "INVALID_REQUEST", "Invalid JSON")
				return
			}
			result, err = s.Wellness.SaveSettings(rpc(pb.WellnessService_SaveSettings_FullMethodName), &pb.WellnessSettings{WaterTargetMl: b.WaterTarget, Version: b.Version})
		} else {
			s.problem(w, requestID, 405, "METHOD_NOT_ALLOWED", "Method not allowed")
			return
		}
		if err != nil {
			s.rpcProblem(w, requestID, err)
			return
		}
		s.json(w, 200, map[string]interface{}{"water_target_ml": result.WaterTargetMl, "version": result.Version})
		return
	}
	id := strings.TrimPrefix(path, "/wellness/records/")
	if path == "/wellness/records" && r.Method == http.MethodGet {
		q := r.URL.Query()
		limit, ok := analysisLimit(w, s, requestID, q.Get("limit"), 50, 50)
		if !ok {
			return
		}
		page, err := s.Wellness.ListRecords(rpc(pb.WellnessService_ListRecords_FullMethodName), &pb.WellnessListRequest{Kind: q.Get("kind"), FromTrackingDay: q.Get("from_tracking_day"), ToTrackingDay: q.Get("to_tracking_day"), Limit: limit, Cursor: q.Get("cursor")})
		if err != nil {
			s.rpcProblem(w, requestID, err)
			return
		}
		items := []interface{}{}
		for _, item := range page.Items {
			items = append(items, wellnessJSON(item))
		}
		s.json(w, 200, map[string]interface{}{"items": items, "page": map[string]interface{}{"next_cursor": nilIfEmpty(page.NextCursor)}})
		return
	}
	if (path == "/wellness/records" && r.Method == http.MethodPost) || (path != "/wellness/records" && r.Method == http.MethodPatch) {
		var b wellnessBody
		if decodeJSON(w, r, &b) != nil {
			s.problem(w, requestID, 400, "INVALID_REQUEST", "Invalid JSON")
			return
		}
		if r.Method == http.MethodPost {
			id = ""
		}
		saved, err := s.Wellness.SaveRecord(rpc(pb.WellnessService_SaveRecord_FullMethodName), &pb.WellnessRecord{Id: id, Version: b.Version, Kind: b.Kind, Amount: b.Amount, Unit: b.Unit, LoggedAt: b.LoggedAt, Name: b.Name, Intensity: b.Intensity, EstimatedKcal: b.EstimatedKcal, Notes: b.Notes})
		if err != nil {
			s.rpcProblem(w, requestID, err)
			return
		}
		code := 200
		if r.Method == http.MethodPost {
			code = 201
		}
		s.json(w, code, wellnessJSON(saved))
		return
	}
	if path != "/wellness/records" && r.Method == http.MethodDelete {
		version, err := strconv.ParseUint(r.URL.Query().Get("version"), 10, 64)
		if err != nil {
			s.problem(w, requestID, 422, "VALIDATION_FAILED", "Version required")
			return
		}
		_, err = s.Wellness.DeleteRecord(rpc(pb.WellnessService_DeleteRecord_FullMethodName), &pb.WellnessDeleteRequest{Id: id, Version: version})
		if err != nil {
			s.rpcProblem(w, requestID, err)
			return
		}
		w.WriteHeader(204)
		return
	}
	s.problem(w, requestID, 405, "METHOD_NOT_ALLOWED", "Method not allowed")
}
