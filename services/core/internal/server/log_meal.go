package server

import (
	"context"

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
	for _, component := range meal.Components {
		items = append(items, &pb.FoodLogItemInput{Position: component.Position,
			FoodType: component.FoodType, ProviderFoodId: component.ProviderFoodId,
			CustomFoodId: component.CustomFoodId, Quantity: component.Quantity,
			FoodState: component.FoodState})
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
