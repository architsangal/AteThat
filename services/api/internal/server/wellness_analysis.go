package server

import (
	"context"
	pb "github.com/architsangal/AteThat/services/api/internal/gen/atethat/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net/http"
)

func maintenanceJSON(p *pb.MaintenancePlan) any {
	history := []any{}
	for _, v := range p.History {
		history = append(history, map[string]any{"revision": v.Revision, "kcal": v.Kcal, "mode": v.Mode, "effective_from_tracking_day": v.EffectiveFromTrackingDay, "created_at": v.CreatedAt, "equation": v.Equation, "height_cm": v.HeightCm, "weight_kg": v.WeightKg, "age": v.Age, "sex": v.Sex})
	}
	return map[string]any{"version": p.Version, "suggested_kcal": p.SuggestedKcal, "equation": p.Equation, "unavailable_reason": p.UnavailableReason, "profile_version": p.ProfileVersion, "inputs": map[string]any{"height_cm": p.Inputs.GetHeightCm(), "weight_kg": p.Inputs.GetWeightKg(), "assumed_height": p.Inputs.GetAssumedHeight(), "assumed_weight": p.Inputs.GetAssumedWeight(), "age": p.Inputs.GetAge(), "sex": p.Inputs.GetSex()}, "history": history, "tracking_day": p.TrackingDay}
}

func (s *Server) wellnessAnalysisRoute(w http.ResponseWriter, r *http.Request, id string, rpc func(string) context.Context, path string) {
	if path == "/wellness/analysis" {
		if r.Method != http.MethodGet {
			s.problem(w, id, 405, "METHOD_NOT_ALLOWED", "Method not allowed")
			return
		}
		p, err := s.Wellness.DailyWellness(rpc(pb.WellnessService_DailyWellness_FullMethodName), &pb.FullAnalysisRequest{From: r.URL.Query().Get("from_tracking_day"), To: r.URL.Query().Get("to_tracking_day")})
		if err != nil {
			if status.Code(err) == codes.InvalidArgument {
				s.problem(w, id, 422, "VALIDATION_FAILED", "Choose a valid Tracking Day range of no more than 366 days, with the start before the end.")
			} else {
				s.rpcProblem(w, id, err)
			}
			return
		}
		items := []any{}
		for _, d := range p.Items {
			items = append(items, map[string]any{"tracking_day": d.TrackingDay, "maintenance_kcal": d.MaintenanceKcal, "food_kcal": d.FoodKcal, "workout_kcal": d.WorkoutKcal, "calorie_deficit": d.CalorieDeficit, "water_ml": d.WaterMl, "workout_count": d.WorkoutCount, "workouts_without_calories": d.WorkoutsWithoutCalories, "workout_minutes": d.WorkoutMinutes, "maintenance_revision": d.MaintenanceRevision})
		}
		s.json(w, 200, map[string]any{"items": items, "water_target_ml": p.WaterTargetMl})
		return
	}
	var p *pb.MaintenancePlan
	var err error
	switch r.Method {
	case http.MethodGet:
		p, err = s.Wellness.GetMaintenance(rpc(pb.WellnessService_GetMaintenance_FullMethodName), &pb.FoodEmpty{})
	case http.MethodPut:
		var body struct {
			Version        uint64 `json:"version"`
			Kcal           string `json:"kcal"`
			Mode           string `json:"mode"`
			ProfileVersion uint64 `json:"profile_version"`
		}
		if decodeJSON(w, r, &body) != nil {
			s.problem(w, id, 400, "INVALID_REQUEST", "Invalid maintenance request")
			return
		}
		p, err = s.Wellness.SaveMaintenance(rpc(pb.WellnessService_SaveMaintenance_FullMethodName), &pb.MaintenanceSaveRequest{Version: body.Version, Kcal: body.Kcal, Mode: body.Mode, ProfileVersion: body.ProfileVersion})
	default:
		s.problem(w, id, 405, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}
	if err != nil {
		if r.Method == http.MethodPut && status.Code(err) == codes.InvalidArgument {
			s.problem(w, id, 422, "VALIDATION_FAILED", "Enter positive maintenance calories and choose a manual or calculated baseline.")
		} else if r.Method == http.MethodPut && status.Code(err) == codes.Aborted {
			s.problem(w, id, 409, "VERSION_CONFLICT", "Your settings or profile estimate changed. Reload the maintenance section and review it before saving.")
		} else if status.Code(err) == codes.ResourceExhausted {
			s.problem(w, id, 409, "MAINTENANCE_HISTORY_FULL", "Your maintenance history is full. Your existing baseline is unchanged. Contact support before adding another revision.")
		} else {
			s.rpcProblem(w, id, err)
		}
		return
	}
	s.json(w, 200, maintenanceJSON(p))
}
