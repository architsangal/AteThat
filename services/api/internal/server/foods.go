package server

import (
	"context"
	"net/http"
	"strconv"

	pb "github.com/architsangal/AteThat/services/api/internal/gen/atethat/v1"
)

func foodJSON(food *pb.FoodResult) interface{} {
	nutrients := make(map[string]interface{}, len(food.Nutrients))
	for key, value := range food.Nutrients {
		entry := map[string]interface{}{"status": value.Status, "unit": value.Unit}
		if value.Status == "KNOWN" {
			entry["amount"] = value.Amount
		}
		nutrients[key] = entry
	}
	return map[string]interface{}{
		"food_reference": map[string]interface{}{"type": "USDA", "provider_food_id": food.ProviderFoodId, "custom_food_id": nil},
		"name":           food.Name, "food_state": food.FoodState,
		"base_quantity": map[string]string{"amount": food.BaseAmount, "unit": food.BaseUnit},
		"nutrients":     nutrients, "data_type": food.DataType, "source_url": food.SourceUrl,
	}
}

func (s *Server) searchFoods(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context) {
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
	result, err := s.Food.SearchUSDA(s.rpcContext(ctx, pb.FoodService_SearchUSDA_FullMethodName, user.UserID, user.SessionFamily),
		&pb.SearchUSDARequest{Query: r.URL.Query().Get("q"), FoodState: r.URL.Query().Get("food_state"),
			Limit: limit, Cursor: r.URL.Query().Get("cursor")})
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	items := make([]interface{}, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, foodJSON(item))
	}
	s.json(w, 200, map[string]interface{}{"items": items,
		"page": map[string]interface{}{"next_cursor": nilIfEmpty(result.NextCursor), "has_more": result.HasMore}})
}

func (s *Server) getProviderFood(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context, id string) {
	user, ok := s.authenticated(r, w, requestID)
	if !ok {
		return
	}
	result, err := s.Food.GetUSDAFood(s.rpcContext(ctx, pb.FoodService_GetUSDAFood_FullMethodName, user.UserID, user.SessionFamily),
		&pb.GetUSDAFoodRequest{ProviderFoodId: id})
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	s.json(w, 200, foodJSON(result))
}
