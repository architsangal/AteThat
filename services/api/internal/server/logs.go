package server

import (
	"context"
	"net/http"
	"strconv"

	pb "github.com/architsangal/AteThat/services/api/internal/gen/atethat/v1"
)

type nutrientBody struct {
	Status string `json:"status"`
	Amount string `json:"amount"`
	Unit   string `json:"unit"`
}
type logGroupBody struct {
	RecipeYield    string `json:"recipe_yield"`
	RecipePortions string `json:"recipe_portions"`
	LoggedAt       string `json:"logged_at"`
	MealCategory   string `json:"meal_category"`
	Label          string `json:"label"`
	Items          []struct {
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
		FoodState             string                  `json:"food_state"`
		UserOverrideNutrition map[string]nutrientBody `json:"user_override_nutrition"`
		ManualFood            *customFoodBody         `json:"manual_food"`
	} `json:"items"`
}

func logNutritionJSON(values map[string]*pb.FoodNutrient) map[string]interface{} {
	result := make(map[string]interface{}, len(values))
	for key, value := range values {
		entry := map[string]interface{}{"status": value.Status, "unit": value.Unit}
		if value.Status == "KNOWN" {
			entry["amount"] = value.Amount
		}
		result[key] = entry
	}
	return result
}

func foodGroupJSON(group *pb.FoodLogGroup) interface{} {
	items := make([]interface{}, 0, len(group.Items))
	for _, item := range group.Items {
		items = append(items, map[string]interface{}{"id": nilIfEmpty(item.Id), "position": item.Position,
			"food_source": map[string]interface{}{"type": item.FoodSource.Type,
				"provider_food_id": nilIfEmpty(item.FoodSource.ProviderFoodId), "custom_food_id": nilIfEmpty(item.FoodSource.CustomFoodId), "name": item.FoodSource.Name},
			"quantity":   map[string]string{"amount": item.Quantity.Amount, "unit": item.Quantity.Unit},
			"food_state": item.FoodState, "provider_nutrition": logNutritionJSON(item.ProviderNutrition),
			"user_override_nutrition": logNutritionJSON(item.UserOverrideNutrition), "final_nutrition": logNutritionJSON(item.FinalNutrition)})
	}
	return map[string]interface{}{"id": nilIfEmpty(group.Id), "root_group_id": nilIfEmpty(group.RootGroupId),
		"revision": group.Revision, "status": group.Status, "logged_at": group.LoggedAt,
		"tracking_day": group.TrackingDay, "timezone_at_logging": group.TimezoneAtLogging,
		"daily_reset_time_at_logging": group.DailyResetTimeAtLogging, "meal_category": group.MealCategory,
		"label": group.Label, "item_count": group.ItemCount, "items": items,
		"aggregate_final_nutrition": logNutritionJSON(group.AggregateFinalNutrition),
		"calculation_version":       group.CalculationVersion, "confirmed_at": nilIfEmpty(group.ConfirmedAt),
		"supersedes_group_id": nilIfEmpty(group.SupersedesGroupId), "correction_reason": group.CorrectionReason}
}

func (s *Server) previewFoodGroup(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context) {
	user, ok := s.authenticated(r, w, requestID)
	if !ok {
		return
	}
	var body logGroupBody
	if decodeJSON(w, r, &body) != nil {
		s.problem(w, requestID, 400, "INVALID_REQUEST", "Invalid JSON request")
		return
	}
	items := make([]*pb.FoodLogItemInput, 0, len(body.Items))
	for _, item := range body.Items {
		overrides := make(map[string]*pb.FoodNutrient, len(item.UserOverrideNutrition))
		for key, value := range item.UserOverrideNutrition {
			overrides[key] = &pb.FoodNutrient{Status: value.Status, Amount: value.Amount, Unit: value.Unit}
		}
		var manual *pb.UpsertCustomFoodRequest
		if item.ManualFood != nil {
			manual = customFoodRequest("", *item.ManualFood)
		}
		items = append(items, &pb.FoodLogItemInput{Position: item.Position, FoodType: item.FoodReference.Type, ManualFood: manual,
			ProviderFoodId: item.FoodReference.ProviderFoodID, CustomFoodId: item.FoodReference.CustomFoodID,
			Quantity:  &pb.FoodQuantity{Amount: item.Quantity.Amount, Unit: item.Quantity.Unit},
			FoodState: item.FoodState, UserOverrideNutrition: overrides})
	}
	result, err := s.FoodLog.Preview(s.rpcContext(ctx, pb.FoodLogService_Preview_FullMethodName, user.UserID, user.SessionFamily),
		&pb.PreviewFoodGroupRequest{LoggedAt: body.LoggedAt, MealCategory: body.MealCategory, Label: body.Label, Items: items, RecipeYield: body.RecipeYield, RecipePortions: body.RecipePortions})
	if err != nil {
		s.foodReviewProblem(w, requestID, err)
		return
	}
	s.json(w, 201, map[string]interface{}{"preview_token": result.PreviewToken, "expires_at": result.ExpiresAt, "group": foodGroupJSON(result.Group)})
}

func (s *Server) confirmFoodGroup(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context) {
	user, ok := s.authenticated(r, w, requestID)
	if !ok {
		return
	}
	key := r.Header.Get("Idempotency-Key")
	if key == "" {
		s.problem(w, requestID, 422, "VALIDATION_FAILED", "Idempotency-Key is required")
		return
	}
	var body struct {
		PreviewToken string `json:"preview_token"`
	}
	if decodeJSON(w, r, &body) != nil {
		s.problem(w, requestID, 400, "INVALID_REQUEST", "Invalid JSON request")
		return
	}
	result, err := s.FoodLog.Confirm(s.rpcContext(ctx, pb.FoodLogService_Confirm_FullMethodName, user.UserID, user.SessionFamily),
		&pb.ConfirmFoodGroupRequest{PreviewToken: body.PreviewToken, IdempotencyKey: key})
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	if result.WasReplayed {
		s.json(w, 200, foodGroupJSON(result))
	} else {
		s.json(w, 201, foodGroupJSON(result))
	}
}

func (s *Server) listFoodGroups(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context) {
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
	result, err := s.FoodLog.ListGroups(s.rpcContext(ctx, pb.FoodLogService_ListGroups_FullMethodName, user.UserID, user.SessionFamily),
		&pb.ListFoodGroupsRequest{TrackingDay: r.URL.Query().Get("tracking_day"), Limit: limit, Cursor: r.URL.Query().Get("cursor")})
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	items := make([]interface{}, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, foodGroupJSON(item))
	}
	s.json(w, 200, map[string]interface{}{"items": items, "page": map[string]interface{}{"next_cursor": nilIfEmpty(result.NextCursor), "has_more": result.HasMore}})
}

func (s *Server) getFoodGroup(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context, id string) {
	user, ok := s.authenticated(r, w, requestID)
	if !ok {
		return
	}
	result, err := s.FoodLog.GetGroup(s.rpcContext(ctx, pb.FoodLogService_GetGroup_FullMethodName, user.UserID, user.SessionFamily), &pb.FoodGroupRequest{Id: id})
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	s.json(w, 200, foodGroupJSON(result))
}

func (s *Server) voidFoodGroup(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context, id string) {
	user, ok := s.authenticated(r, w, requestID)
	if !ok {
		return
	}
	_, err := s.FoodLog.VoidGroup(s.rpcContext(ctx, pb.FoodLogService_VoidGroup_FullMethodName, user.UserID, user.SessionFamily), &pb.FoodGroupRequest{Id: id})
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) getFoodLogItem(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context, id string) {
	user, ok := s.authenticated(r, w, requestID)
	if !ok {
		return
	}
	item, err := s.FoodLog.GetLogItem(s.rpcContext(ctx, pb.FoodLogService_GetLogItem_FullMethodName, user.UserID, user.SessionFamily), &pb.FoodLogItemRequest{Id: id})
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	s.json(w, 200, map[string]interface{}{"id": item.Id, "position": item.Position,
		"food_source": map[string]interface{}{"type": item.FoodSource.Type, "provider_food_id": nilIfEmpty(item.FoodSource.ProviderFoodId),
			"custom_food_id": nilIfEmpty(item.FoodSource.CustomFoodId), "name": item.FoodSource.Name},
		"quantity": map[string]string{"amount": item.Quantity.Amount, "unit": item.Quantity.Unit}, "food_state": item.FoodState,
		"provider_nutrition": logNutritionJSON(item.ProviderNutrition), "user_override_nutrition": logNutritionJSON(item.UserOverrideNutrition),
		"final_nutrition": logNutritionJSON(item.FinalNutrition)})
}

func (s *Server) correctFoodLogItem(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context, id string) {
	user, ok := s.authenticated(r, w, requestID)
	if !ok {
		return
	}
	var body struct {
		Quantity struct {
			Amount string `json:"amount"`
			Unit   string `json:"unit"`
		} `json:"quantity"`
		Reason string `json:"reason"`
	}
	if decodeJSON(w, r, &body) != nil {
		s.problem(w, requestID, 400, "INVALID_REQUEST", "Invalid JSON request")
		return
	}
	result, err := s.FoodLog.CorrectItem(s.rpcContext(ctx, pb.FoodLogService_CorrectItem_FullMethodName, user.UserID, user.SessionFamily),
		&pb.CorrectFoodLogItemRequest{Id: id, Quantity: &pb.FoodQuantity{Amount: body.Quantity.Amount, Unit: body.Quantity.Unit}, Reason: body.Reason})
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	s.json(w, 201, map[string]interface{}{"preview_token": result.PreviewToken, "expires_at": result.ExpiresAt, "group": foodGroupJSON(result.Group)})
}

func (s *Server) voidFoodLogItem(w http.ResponseWriter, r *http.Request, requestID string, ctx context.Context, id string) {
	user, ok := s.authenticated(r, w, requestID)
	if !ok {
		return
	}
	_, err := s.FoodLog.VoidItem(s.rpcContext(ctx, pb.FoodLogService_VoidItem_FullMethodName, user.UserID, user.SessionFamily), &pb.FoodLogItemRequest{Id: id})
	if err != nil {
		s.rpcProblem(w, requestID, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
