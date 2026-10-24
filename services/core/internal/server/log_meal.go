package server

import (
	"context"
	"github.com/architsangal/AteThat/services/core/internal/foods"

	pb "github.com/architsangal/AteThat/services/core/internal/gen/atethat/v1"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (s *Server) PreviewMeal(ctx context.Context, req *pb.PreviewMealRequest) (*pb.FoodGroupPreview, error) {
	userID, err := Principal(ctx)
	if err != nil {
		return nil, err
	}
	meal, err := s.GetMeal(ctx, &pb.MealRequest{Id: req.MealId})
	if err != nil {
		return nil, err
	}
	items := make([]*pb.FoodLogItemInput, 0, len(meal.Components))
	if len(req.Overrides) > len(meal.Components) {
		return nil, invalid("too many recipe overrides")
	}
	overrides := map[uint32]map[string]*pb.FoodNutrient{}
	for _, edit := range req.Overrides {
		if edit == nil || int(edit.Position) >= len(meal.Components) {
			return nil, invalid("invalid ingredient position")
		}
		if _, exists := overrides[edit.Position]; exists {
			return nil, invalid("duplicate ingredient override")
		}
		overrides[edit.Position] = edit.Nutrients
	}
	for _, component := range meal.Components {
		quantity := component.Quantity
		if req.Servings != "" {
			yield := meal.RecipeServings
			if yield == "" {
				yield = "1"
			} // Older saved meals represent one complete serving.
			amount, err := foods.Scale(component.Quantity.Amount, req.Servings, yield)
			if err != nil || amount == "0" {
				return nil, invalid("invalid recipe portion")
			}
			quantity = &pb.FoodQuantity{Amount: amount, Unit: component.Quantity.Unit}
		}
		items = append(items, &pb.FoodLogItemInput{Position: component.Position,
			FoodType: component.FoodType, ProviderFoodId: component.ProviderFoodId,
			CustomFoodId: component.CustomFoodId, Quantity: quantity,
			FoodState: component.FoodState, UserOverrideNutrition: overrides[component.Position]})
	}
	preview, err := s.Preview(ctx, &pb.PreviewFoodGroupRequest{LoggedAt: req.LoggedAt,
		MealCategory: req.MealCategory, Label: req.Label, Items: items})
	if err != nil {
		return nil, err
	}
	previewID, _, err := parsePreviewToken(preview.PreviewToken)
	if err != nil {
		return nil, err
	}
	mealID, _ := primitive.ObjectIDFromHex(meal.Id)
	_, err = s.DB.Collection("food_log_previews").UpdateOne(ctx,
		bson.M{"_id": previewID, "user_id": userID, "status": "OPEN"},
		bson.M{"$set": bson.M{"source_type": "SAVED_MEAL", "source_meal_id": mealID}})
	if err != nil {
		return nil, internal()
	}
	return preview, nil
}
