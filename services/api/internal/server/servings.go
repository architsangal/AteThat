package server

import (
	"context"
	"net/http"
	"strconv"

	pb "github.com/architsangal/AteThat/services/api/internal/gen/atethat/v1"
)

type servingBody struct {
	Version       uint64 `json:"version"`
	Label         string `json:"label"`
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
}

func servingRequest(id string, body servingBody) *pb.UpsertServingRequest {
	return &pb.UpsertServingRequest{Id: id, Version: body.Version, Label: body.Label,
		FoodType: body.FoodReference.Type, ProviderFoodId: body.FoodReference.ProviderFoodID,
		CustomFoodId: body.FoodReference.CustomFoodID,
		Quantity:     &pb.FoodQuantity{Amount: body.Quantity.Amount, Unit: body.Quantity.Unit}, FoodState: body.FoodState}
}

func servingJSON(item *pb.ServingDefinition) interface{} {
	return map[string]interface{}{"id": item.Id, "version": item.Version, "label": item.Label,
		"food_reference": map[string]interface{}{"type": item.FoodType, "provider_food_id": nilIfEmpty(item.ProviderFoodId), "custom_food_id": nilIfEmpty(item.CustomFoodId)},
		"quantity":       map[string]string{"amount": item.Quantity.Amount, "unit": item.Quantity.Unit},
		"food_state":     item.FoodState, "created_at": item.CreatedAt, "updated_at": item.UpdatedAt}
}

func (s *Server) createServing(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context) {
	user, ok := s.authenticated(r, w, requestID)
	if !ok {
		return
	}
	var body servingBody
	if decodeJSON(w, r, &body) != nil {
		s.problem(w, requestID, 400, "INVALID_REQUEST", "Invalid JSON request")
		return
	}
	item, err := s.Food.CreateServing(s.rpcContext(ctx, pb.FoodService_CreateServing_FullMethodName, user.UserID, user.SessionFamily), servingRequest("", body))
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	w.Header().Set("Location", "/api/v1/servings/"+item.Id)
	s.json(w, 201, servingJSON(item))
}

func (s *Server) updateServing(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context, id string) {
	user, ok := s.authenticated(r, w, requestID)
	if !ok {
		return
	}
	var body servingBody
	if decodeJSON(w, r, &body) != nil {
		s.problem(w, requestID, 400, "INVALID_REQUEST", "Invalid JSON request")
		return
	}
	item, err := s.Food.UpdateServing(s.rpcContext(ctx, pb.FoodService_UpdateServing_FullMethodName, user.UserID, user.SessionFamily), servingRequest(id, body))
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	s.json(w, 200, servingJSON(item))
}

func (s *Server) getServing(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context, id string) {
	user, ok := s.authenticated(r, w, requestID)
	if !ok {
		return
	}
	item, err := s.Food.GetServing(s.rpcContext(ctx, pb.FoodService_GetServing_FullMethodName, user.UserID, user.SessionFamily), &pb.ServingRequest{Id: id})
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	s.json(w, 200, servingJSON(item))
}

func (s *Server) listServings(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context) {
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
	page, err := s.Food.ListServings(s.rpcContext(ctx, pb.FoodService_ListServings_FullMethodName, user.UserID, user.SessionFamily),
		&pb.ListServingsRequest{Limit: limit, Cursor: r.URL.Query().Get("cursor")})
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	items := make([]interface{}, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, servingJSON(item))
	}
	s.json(w, 200, map[string]interface{}{"items": items, "page": map[string]interface{}{"next_cursor": nilIfEmpty(page.NextCursor), "has_more": page.HasMore}})
}

func (s *Server) deleteServing(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context, id string) {
	user, ok := s.authenticated(r, w, requestID)
	if !ok {
		return
	}
	_, err := s.Food.DeleteServing(s.rpcContext(ctx, pb.FoodService_DeleteServing_FullMethodName, user.UserID, user.SessionFamily), &pb.ServingRequest{Id: id})
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
