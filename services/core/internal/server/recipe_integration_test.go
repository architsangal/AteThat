package server

import (
	"context"
	pb "github.com/architsangal/AteThat/services/core/internal/gen/atethat/v1"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"os"
	"testing"
	"time"
)

func TestRecipePortionsAndSnapshotsIntegration(t *testing.T) {
	uri := os.Getenv("ATETHAT_TEST_MONGO_URI")
	if uri == "" {
		t.Skip("local replica set required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Disconnect(context.Background())
	db := client.Database("atethat_recipe_test_" + primitive.NewObjectID().Hex())
	defer db.Drop(context.Background())
	s := &Server{DB: db, Client: client}
	user := primitive.NewObjectID()
	owner := context.WithValue(ctx, principalKey{}, user)
	other := context.WithValue(ctx, principalKey{}, primitive.NewObjectID())
	if _, err := db.Collection("user_profiles").InsertOne(ctx, profileDoc{UserID: user, Timezone: "Asia/Kolkata"}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Collection("user_preferences").InsertOne(ctx, preferencesDoc{UserID: user, DailyResetTime: "04:00"}); err != nil {
		t.Fatal(err)
	}
	food, err := s.CreateCustomFood(owner, &pb.UpsertCustomFoodRequest{Name: "Test ingredient", FoodState: "RAW", BaseQuantity: &pb.FoodQuantity{Amount: "100", Unit: "g"}, Nutrients: map[string]*pb.FoodNutrient{"ENERGY": {Status: "KNOWN", Amount: "200", Unit: "kcal"}, "IRON": {Status: "UNKNOWN", Unit: "mg"}}})
	if err != nil {
		t.Fatal(err)
	}
	input := &pb.UpsertMealRequest{Name: "Test recipe", RecipeServings: "4", Preparation: "Cook together", Components: []*pb.MealComponent{{Position: 0, FoodType: "CUSTOM", CustomFoodId: food.Id, Quantity: &pb.FoodQuantity{Amount: "300", Unit: "g"}, FoodState: "RAW"}}}
	recipe, err := s.CreateMeal(owner, input)
	if err != nil {
		t.Fatal(err)
	}
	request := &pb.PreviewMealRequest{MealId: recipe.Id, Servings: "1.5", LoggedAt: time.Now().UTC().Format(time.RFC3339), MealCategory: "LUNCH", Label: recipe.Name}
	preview, err := s.PreviewMeal(owner, request)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Group.Items[0].Quantity.Amount != "112.5" || preview.Group.AggregateFinalNutrition["ENERGY"].Amount != "225" || preview.Group.AggregateFinalNutrition["IRON"].Status != "UNKNOWN" {
		t.Fatalf("bad portion: %+v", preview.Group)
	}
	unsaved, err := s.Preview(owner, &pb.PreviewFoodGroupRequest{LoggedAt: request.LoggedAt, MealCategory: "OTHER", RecipeYield: "4", RecipePortions: "1.5", Items: []*pb.FoodLogItemInput{{Position: 0, FoodType: "CUSTOM", CustomFoodId: food.Id, FoodState: "RAW", Quantity: &pb.FoodQuantity{Amount: "300", Unit: "g"}}}})
	if err != nil || unsaved.Group.AggregateFinalNutrition["ENERGY"].Amount != "225" {
		t.Fatal("unsaved recipe scaling differs from saved portion", err)
	}
	if _, err := s.PreviewMeal(other, request); status.Code(err) != codes.NotFound {
		t.Fatalf("foreign recipe visible: %v", err)
	}
	request.Overrides = []*pb.RecipeNutrientOverride{{Position: 0, Nutrients: map[string]*pb.FoodNutrient{"ENERGY": {Status: "KNOWN", Amount: "180", Unit: "kcal"}}}}
	overridden, err := s.PreviewMeal(owner, request)
	if err != nil || overridden.Group.AggregateFinalNutrition["ENERGY"].Amount != "180" || overridden.Group.Items[0].ProviderNutrition["ENERGY"].Amount != "225" {
		t.Fatal("override loses provider value", overridden, err)
	}
	for _, edits := range [][]*pb.RecipeNutrientOverride{
		{{Position: 2}}, {{Position: 0}, {Position: 0}}, {{Position: 0, Nutrients: map[string]*pb.FoodNutrient{"ENERGY": {Status: "KNOWN", Amount: "-1", Unit: "kcal"}}}},
		{{Position: 0, Nutrients: map[string]*pb.FoodNutrient{"ENERGY": {Status: "KNOWN", Amount: "1", Unit: "kg"}}}},
	} {
		request.Overrides = edits
		if _, err := s.PreviewMeal(owner, request); status.Code(err) != codes.InvalidArgument {
			t.Fatal("invalid override accepted", edits, err)
		}
	}
	request.Overrides = nil
	confirmed, err := s.Confirm(owner, &pb.ConfirmFoodGroupRequest{PreviewToken: preview.PreviewToken, IdempotencyKey: "recipe-test-confirm"})
	if err != nil {
		t.Fatal(err)
	}
	input.Id, input.Version, input.RecipeServings = recipe.Id, recipe.Version, "8"
	if _, err := s.UpdateMeal(owner, input); err != nil {
		t.Fatal(err)
	}
	var stored foodLogGroupDoc
	id, _ := primitive.ObjectIDFromHex(confirmed.Id)
	if err := db.Collection("food_log_groups").FindOne(ctx, bson.M{"_id": id}).Decode(&stored); err != nil {
		t.Fatal(err)
	}
	if stored.Aggregate["ENERGY"].Amount.String() != "225" {
		t.Fatal("recipe edit mutated confirmed history")
	}
	for _, value := range []string{"0", "-1", "NaN", "1e3"} {
		request.Servings = value
		if _, err := s.PreviewMeal(owner, request); status.Code(err) != codes.InvalidArgument {
			t.Fatalf("invalid serving accepted: %s %v", value, err)
		}
	}
	input.Id, input.Version, input.RecipeServings = "", 0, ""
	meal, err := s.CreateMeal(owner, input)
	if err != nil {
		t.Fatal(err)
	}
	request.MealId, request.Servings = meal.Id, ""
	whole, err := s.PreviewMeal(owner, request)
	if err != nil || whole.Group.Items[0].Quantity.Amount != "300" {
		t.Fatal("legacy meal changed", err)
	}
}
