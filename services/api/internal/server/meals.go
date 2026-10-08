package server

import (
	"context"
	"net/http"
	"strconv"

	pb "github.com/architsangal/AteThat/services/api/internal/gen/atethat/v1"
)

type mealBody struct {
	Version    uint64 `json:"version"`
	Name       string `json:"name"`
	Components []struct {
		Position      uint32 `json:"position"`
		FoodReference struct {
			Type           string `json:"type"`
			ProviderFoodID string `json:"provider_food_id"`
			CustomFoodID   string `json:"custom_food_id"`
		} `json:"food_reference"`
		Quantity struct {
			Amount string `json:"amount"`
			Unit   string `json:"unit"`
		} `json:"quantity"`
		FoodState string `json:"food_state"`
	} `json:"components"`
}

func mealRequest(id string, body mealBody) *pb.UpsertMealRequest {
	components := make([]*pb.MealComponent, 0, len(body.Components))
	for _, item := range body.Components {
		components = append(components, &pb.MealComponent{Position: item.Position,
			FoodType: item.FoodReference.Type, ProviderFoodId: item.FoodReference.ProviderFoodID, CustomFoodId: item.FoodReference.CustomFoodID,
			Quantity: &pb.FoodQuantity{Amount: item.Quantity.Amount, Unit: item.Quantity.Unit}, FoodState: item.FoodState})
	}
	return &pb.UpsertMealRequest{Id: id, Version: body.Version, Name: body.Name, Components: components}
}

func mealJSON(meal *pb.Meal) interface{} {
	components := make([]interface{}, 0, len(meal.Components))
	for _, item := range meal.Components {
		components = append(components, map[string]interface{}{"position": item.Position,
			"food_reference": map[string]interface{}{"type": item.FoodType, "provider_food_id": nilIfEmpty(item.ProviderFoodId), "custom_food_id": nilIfEmpty(item.CustomFoodId)},
			"quantity":       map[string]string{"amount": item.Quantity.Amount, "unit": item.Quantity.Unit}, "food_state": item.FoodState})
	}
	return map[string]interface{}{"id": meal.Id, "version": meal.Version, "name": meal.Name, "components": components,
		"created_at": meal.CreatedAt, "updated_at": meal.UpdatedAt}
}

func (s *Server) createMeal(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context) {
	user, ok := s.authenticated(r, w, requestID)
	if !ok {
		return
	}
	var body mealBody
	if decodeJSON(w, r, &body) != nil {
		s.problem(w, requestID, 400, "INVALID_REQUEST", "Invalid JSON request")
		return
	}
	meal, err := s.Food.CreateMeal(s.rpcContext(ctx, pb.FoodService_CreateMeal_FullMethodName, user.UserID, user.SessionFamily), mealRequest("", body))
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	w.Header().Set("Location", "/api/v1/meals/"+meal.Id)
	s.json(w, 201, mealJSON(meal))
}

func (s *Server) updateMeal(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context, id string) {
	user, ok := s.authenticated(r, w, requestID)
	if !ok {
		return
	}
	var body mealBody
	if decodeJSON(w, r, &body) != nil {
		s.problem(w, requestID, 400, "INVALID_REQUEST", "Invalid JSON request")
		return
	}
	meal, err := s.Food.UpdateMeal(s.rpcContext(ctx, pb.FoodService_UpdateMeal_FullMethodName, user.UserID, user.SessionFamily), mealRequest(id, body))
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	s.json(w, 200, mealJSON(meal))
}

func (s *Server) getMeal(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context, id string) {
	user, ok := s.authenticated(r, w, requestID)
	if !ok {
		return
	}
	meal, err := s.Food.GetMeal(s.rpcContext(ctx, pb.FoodService_GetMeal_FullMethodName, user.UserID, user.SessionFamily), &pb.MealRequest{Id: id})
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	s.json(w, 200, mealJSON(meal))
}

func (s *Server) listMeals(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context) {
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
	page, err := s.Food.ListMeals(s.rpcContext(ctx, pb.FoodService_ListMeals_FullMethodName, user.UserID, user.SessionFamily), &pb.ListMealsRequest{Limit: limit, Cursor: r.URL.Query().Get("cursor")})
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	items := make([]interface{}, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, mealJSON(item))
	}
	s.json(w, 200, map[string]interface{}{"items": items, "page": map[string]interface{}{"next_cursor": nilIfEmpty(page.NextCursor), "has_more": page.HasMore}})
}

func (s *Server) deleteMeal(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context, id string) {
	user, ok := s.authenticated(r, w, requestID)
	if !ok {
		return
	}
	_, err := s.Food.DeleteMeal(s.rpcContext(ctx, pb.FoodService_DeleteMeal_FullMethodName, user.UserID, user.SessionFamily), &pb.MealRequest{Id: id})
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) previewMeal(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context, id string) {
	user, ok := s.authenticated(r, w, requestID)
	if !ok {
		return
	}
	var body struct {
		LoggedAt     string `json:"logged_at"`
		MealCategory string `json:"meal_category"`
		Label        string `json:"label"`
	}
	if decodeJSON(w, r, &body) != nil {
		s.problem(w, requestID, 400, "INVALID_REQUEST", "Invalid JSON request")
		return
	}
	result, err := s.FoodLog.PreviewMeal(s.rpcContext(ctx, pb.FoodLogService_PreviewMeal_FullMethodName, user.UserID, user.SessionFamily),
		&pb.PreviewMealRequest{MealId: id, LoggedAt: body.LoggedAt, MealCategory: body.MealCategory, Label: body.Label})
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	s.json(w, 201, map[string]interface{}{"preview_token": result.PreviewToken, "expires_at": result.ExpiresAt, "group": foodGroupJSON(result.Group)})
}
