package server

import (
	"context"
	"os"
	"testing"
	"time"

	pb "github.com/architsangal/AteThat/services/core/internal/gen/atethat/v1"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestCustomFoodOwnershipIntegration(t *testing.T) {
	uri := os.Getenv("ATETHAT_TEST_MONGO_URI")
	if uri == "" {
		t.Skip("set ATETHAT_TEST_MONGO_URI to test ownership against a local replica set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Disconnect(context.Background())
	db := client.Database("atethat_ownership_test_" + primitive.NewObjectID().Hex())
	defer db.Drop(context.Background())
	server := &Server{DB: db, Client: client}
	owner := context.WithValue(ctx, principalKey{}, primitive.NewObjectID())
	other := context.WithValue(ctx, principalKey{}, primitive.NewObjectID())
	input := &pb.UpsertCustomFoodRequest{Name: "Private test food", FoodState: "PREPARED",
		BaseQuantity: &pb.FoodQuantity{Amount: "100", Unit: "g"},
		Nutrients:    map[string]*pb.FoodNutrient{"ENERGY": {Status: "KNOWN", Amount: "100", Unit: "kcal"}}}
	created, err := server.CreateCustomFood(owner, input)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := server.GetCustomFood(other, &pb.CustomFoodRequest{Id: created.Id}); status.Code(err) != codes.NotFound {
		t.Fatalf("foreign read should be not found: %v", err)
	}
	if _, err := server.UpdateCustomFood(other, &pb.UpsertCustomFoodRequest{Id: created.Id, Version: created.Version,
		Name: input.Name, FoodState: input.FoodState, BaseQuantity: input.BaseQuantity, Nutrients: input.Nutrients}); status.Code(err) != codes.Aborted {
		t.Fatalf("foreign update should be rejected: %v", err)
	}
	if _, err := server.DeleteCustomFood(other, &pb.CustomFoodRequest{Id: created.Id}); status.Code(err) != codes.NotFound {
		t.Fatalf("foreign delete should be rejected: %v", err)
	}
	if _, err := server.GetCustomFood(owner, &pb.CustomFoodRequest{Id: created.Id}); err != nil {
		t.Fatalf("foreign calls affected owner record: %v", err)
	}
	servingInput := &pb.UpsertServingRequest{Label: "One small bowl", FoodType: "CUSTOM", CustomFoodId: created.Id,
		FoodState: "PREPARED", Quantity: &pb.FoodQuantity{Amount: "45", Unit: "g"}}
	serving, err := server.CreateServing(owner, servingInput)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := server.GetServing(other, &pb.ServingRequest{Id: serving.Id}); status.Code(err) != codes.NotFound {
		t.Fatalf("foreign serving read should be not found: %v", err)
	}
	if _, err := server.DeleteServing(other, &pb.ServingRequest{Id: serving.Id}); status.Code(err) != codes.NotFound {
		t.Fatalf("foreign serving delete should be not found: %v", err)
	}
	servingInput.Id, servingInput.Version, servingInput.Label = serving.Id, serving.Version, "One medium bowl"
	updated, err := server.UpdateServing(owner, servingInput)
	if err != nil || updated.Version != 2 || updated.Label != "One medium bowl" {
		t.Fatalf("serving update failed: %+v %v", updated, err)
	}
}
