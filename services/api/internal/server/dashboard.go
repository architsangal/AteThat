package server

import (
	"context"
	"net/http"
	"net/url"

	pb "github.com/architsangal/AteThat/services/api/internal/gen/atethat/v1"
)

func (s *Server) getDashboard(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context) {
	user, ok := s.authenticated(r, w, requestID)
	if !ok {
		return
	}
	result, err := s.Dashboard.GetDashboard(s.rpcContext(ctx, pb.DashboardService_GetDashboard_FullMethodName, user.UserID, user.SessionFamily),
		&pb.DashboardRequest{TrackingDay: r.URL.Query().Get("tracking_day")})
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	groups := make([]interface{}, 0, len(result.FoodLogGroups))
	for _, group := range result.FoodLogGroups {
		groups = append(groups, foodGroupJSON(group))
	}
	progress := make(map[string]interface{}, len(result.Progress))
	for key, value := range result.Progress {
		progress[key] = map[string]string{
			"consumed_amount": value.ConsumedAmount, "target_amount": value.TargetAmount,
			"unit": value.Unit, "percent": value.Percent}
	}
	s.json(w, 200, map[string]interface{}{"tracking_day": result.TrackingDay,
		"timezone": result.Timezone, "daily_reset_time": result.DailyResetTime,
		"nutrition": map[string]interface{}{"totals": logNutritionJSON(result.Totals),
			"targets": logNutritionJSON(result.Targets), "progress": progress},
		"food_log_groups": map[string]interface{}{"items": groups, "has_more": result.FoodGroupsHasMore,
			"history_href": "/api/v1/food-log-groups?tracking_day=" + url.QueryEscape(result.TrackingDay)}})
}
