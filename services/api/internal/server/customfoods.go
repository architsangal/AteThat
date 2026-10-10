package server

import (
	"context"
	"net/http"
	"strconv"

	pb "github.com/architsangal/AteThat/services/api/internal/gen/atethat/v1"
)

type customFoodBody struct {
	Version      uint64 `json:"version"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	FoodState    string `json:"food_state"`
	BaseQuantity struct {
		Amount string `json:"amount"`
		Unit   string `json:"unit"`
	} `json:"base_quantity"`
	Nutrients map[string]struct {
		Status string `json:"status"`
		Amount string `json:"amount"`
		Unit   string `json:"unit"`
	} `json:"nutrients"`
}

func customFoodRequest(id string, body customFoodBody) *pb.UpsertCustomFoodRequest {
	nutrients := make(map[string]*pb.FoodNutrient, len(body.Nutrients))
	for key, value := range body.Nutrients {
		nutrients[key] = &pb.FoodNutrient{Status: value.Status, Amount: value.Amount, Unit: value.Unit}
	}
	return &pb.UpsertCustomFoodRequest{Id: id, Version: body.Version, Name: body.Name,
		Description: body.Description, FoodState: body.FoodState,
		BaseQuantity: &pb.FoodQuantity{Amount: body.BaseQuantity.Amount, Unit: body.BaseQuantity.Unit},
		Nutrients:    nutrients}
}

func customFoodJSON(food *pb.CustomFood) interface{} {
	nutrients := make(map[string]interface{}, len(food.Nutrients))
	for key, value := range food.Nutrients {
		entry := map[string]interface{}{"status": value.Status, "unit": value.Unit}
		if value.Status == "KNOWN" {
			entry["amount"] = value.Amount
		}
		nutrients[key] = entry
	}
	return map[string]interface{}{"id": food.Id, "version": food.Version, "name": food.Name,
		"description": food.Description, "food_state": food.FoodState,
		"base_quantity": map[string]string{"amount": food.BaseQuantity.Amount, "unit": food.BaseQuantity.Unit},
		"nutrients":     nutrients, "created_at": food.CreatedAt, "updated_at": food.UpdatedAt}
}

func (s *Server) createCustomFood(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context) {
	user, ok := s.authenticated(r, w, requestID)
	if !ok {
		return
	}
	var body customFoodBody
	if decodeJSON(w, r, &body) != nil {
		s.problem(w, requestID, 400, "INVALID_REQUEST", "Invalid JSON request")
		return
	}
	result, err := s.Food.CreateCustomFood(s.rpcContext(ctx, pb.FoodService_CreateCustomFood_FullMethodName, user.UserID, user.SessionFamily), customFoodRequest("", body))
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	w.Header().Set("Location", "/api/v1/foods/custom/"+result.Id)
	s.json(w, 201, customFoodJSON(result))
}

func (s *Server) updateCustomFood(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context, id string) {
	user, ok := s.authenticated(r, w, requestID)
	if !ok {
		return
	}
	var body customFoodBody
	if decodeJSON(w, r, &body) != nil {
		s.problem(w, requestID, 400, "INVALID_REQUEST", "Invalid JSON request")
		return
	}
	result, err := s.Food.UpdateCustomFood(s.rpcContext(ctx, pb.FoodService_UpdateCustomFood_FullMethodName, user.UserID, user.SessionFamily), customFoodRequest(id, body))
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	s.json(w, 200, customFoodJSON(result))
}

func (s *Server) getCustomFood(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context, id string) {
	user, ok := s.authenticated(r, w, requestID)
	if !ok {
		return
	}
	result, err := s.Food.GetCustomFood(s.rpcContext(ctx, pb.FoodService_GetCustomFood_FullMethodName, user.UserID, user.SessionFamily), &pb.CustomFoodRequest{Id: id})
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	s.json(w, 200, customFoodJSON(result))
}

func (s *Server) listCustomFoods(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context) {
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
	result, err := s.Food.ListCustomFoods(s.rpcContext(ctx, pb.FoodService_ListCustomFoods_FullMethodName, user.UserID, user.SessionFamily), &pb.ListCustomFoodsRequest{Limit: limit, Cursor: r.URL.Query().Get("cursor")})
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	items := make([]interface{}, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, customFoodJSON(item))
	}
	s.json(w, 200, map[string]interface{}{"items": items, "page": map[string]interface{}{"next_cursor": nilIfEmpty(result.NextCursor), "has_more": result.HasMore}})
}

func (s *Server) deleteCustomFood(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context, id string) {
	user, ok := s.authenticated(r, w, requestID)
	if !ok {
		return
	}
	_, err := s.Food.DeleteCustomFood(s.rpcContext(ctx, pb.FoodService_DeleteCustomFood_FullMethodName, user.UserID, user.SessionFamily), &pb.CustomFoodRequest{Id: id})
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
