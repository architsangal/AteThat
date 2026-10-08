package server

import (
	"context"
	"net/http"
	"strconv"

	pb "github.com/architsangal/AteThat/services/api/internal/gen/atethat/v1"
)

func favouriteJSON(item *pb.FoodFavourite) interface{} {
	return map[string]interface{}{"food_key": item.FoodKey, "food_reference": map[string]interface{}{
		"type": item.FoodType, "provider_food_id": nilIfEmpty(item.ProviderFoodId), "custom_food_id": nilIfEmpty(item.CustomFoodId)},
		"name": item.Name, "created_at": item.CreatedAt}
}

func (s *Server) addFavourite(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context) {
	user, ok := s.authenticated(r, w, requestID)
	if !ok {
		return
	}
	var body struct {
		FoodReference struct {
			Type           string `json:"type"`
			ProviderFoodID string `json:"provider_food_id"`
			CustomFoodID   string `json:"custom_food_id"`
		} `json:"food_reference"`
	}
	if decodeJSON(w, r, &body) != nil {
		s.problem(w, requestID, 400, "INVALID_REQUEST", "Invalid JSON request")
		return
	}
	item, err := s.Food.AddFavourite(s.rpcContext(ctx, pb.FoodService_AddFavourite_FullMethodName, user.UserID, user.SessionFamily),
		&pb.FavouriteRequest{FoodType: body.FoodReference.Type, ProviderFoodId: body.FoodReference.ProviderFoodID, CustomFoodId: body.FoodReference.CustomFoodID})
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	s.json(w, 200, favouriteJSON(item))
}

func (s *Server) listFavourites(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context) {
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
	page, err := s.Food.ListFavourites(s.rpcContext(ctx, pb.FoodService_ListFavourites_FullMethodName, user.UserID, user.SessionFamily),
		&pb.ListFavouritesRequest{Limit: limit, Cursor: r.URL.Query().Get("cursor")})
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	items := make([]interface{}, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, favouriteJSON(item))
	}
	s.json(w, 200, map[string]interface{}{"items": items, "page": map[string]interface{}{"next_cursor": nilIfEmpty(page.NextCursor), "has_more": page.HasMore}})
}

func (s *Server) deleteFavourite(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context, key string) {
	user, ok := s.authenticated(r, w, requestID)
	if !ok {
		return
	}
	_, err := s.Food.DeleteFavourite(s.rpcContext(ctx, pb.FoodService_DeleteFavourite_FullMethodName, user.UserID, user.SessionFamily), &pb.FavouriteRequest{FoodKey: key})
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listUsedFoods(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context, sort string) {
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
	page, err := s.Food.ListUsedFoods(s.rpcContext(ctx, pb.FoodService_ListUsedFoods_FullMethodName, user.UserID, user.SessionFamily),
		&pb.ListUsedFoodsRequest{Sort: sort, From: r.URL.Query().Get("from"), To: r.URL.Query().Get("to"), Limit: limit, Cursor: r.URL.Query().Get("cursor")})
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	items := make([]interface{}, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, map[string]interface{}{"food_key": item.FoodKey,
			"food_reference": map[string]interface{}{"type": item.FoodType, "provider_food_id": nilIfEmpty(item.ProviderFoodId), "custom_food_id": nilIfEmpty(item.CustomFoodId)},
			"name":           item.Name, "confirmed_log_count": item.ConfirmedLogCount, "last_logged_at": item.LastLoggedAt})
	}
	s.json(w, 200, map[string]interface{}{"items": items, "page": map[string]interface{}{"next_cursor": nilIfEmpty(page.NextCursor), "has_more": page.HasMore}})
}
