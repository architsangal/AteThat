package server

import (
	"context"
	"strconv"
	"strings"
	"time"

	pb "github.com/architsangal/AteThat/services/core/internal/gen/atethat/v1"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) mealInput(ctx context.Context, userID primitive.ObjectID, req *pb.UpsertMealRequest) (mealDoc, error) {
	name := strings.TrimSpace(req.Name)
	if req.RecipeServings != "" {
		if _, ok := positiveDecimal(req.RecipeServings); !ok {
			return mealDoc{}, invalid("recipe servings must be a positive decimal")
		}
	}
	if len(req.Preparation) > 4000 {
		return mealDoc{}, invalid("preparation must be at most 4000 characters")
	}
	if len(name) < 2 || len(name) > 120 || len(req.Components) < 1 || len(req.Components) > 50 {
		return mealDoc{}, invalid("meal requires a name and 1–50 components")
	}
	components := make([]mealComponentDoc, 0, len(req.Components))
	for index, input := range req.Components {
		if input == nil || int(input.Position) != index || input.Quantity == nil {
			return mealDoc{}, invalid("meal positions must be consecutive")
		}
		amount, ok := positiveDecimal(input.Quantity.Amount)
		if !ok || (input.Quantity.Unit != "g" && input.Quantity.Unit != "ml") {
			return mealDoc{}, invalid("invalid component quantity")
		}
		if input.FoodState != "RAW" && input.FoodState != "COOKED" && input.FoodState != "PREPARED" && input.FoodState != "UNKNOWN" {
			return mealDoc{}, invalid("invalid food state")
		}
		if input.FoodType == "USDA" {
			id, err := strconv.ParseInt(input.ProviderFoodId, 10, 64)
			if err != nil || id < 1 || input.CustomFoodId != "" {
				return mealDoc{}, invalid("invalid USDA food reference")
			}
		} else if input.FoodType == "CUSTOM" {
			id, err := primitive.ObjectIDFromHex(input.CustomFoodId)
			if err != nil || input.ProviderFoodId != "" {
				return mealDoc{}, invalid("invalid custom food reference")
			}
			if s.DB.Collection("custom_foods").FindOne(ctx, bson.M{"_id": id, "user_id": userID, "deleted_at": bson.M{"$exists": false}}).Err() != nil {
				return mealDoc{}, status.Error(codes.NotFound, "custom food not found")
			}
		} else {
			return mealDoc{}, invalid("invalid food reference type")
		}
		components = append(components, mealComponentDoc{Position: index, FoodType: input.FoodType,
			ProviderFoodID: input.ProviderFoodId, CustomFoodID: input.CustomFoodId,
			Quantity: quantityDoc{Amount: amount, Unit: input.Quantity.Unit}, FoodState: input.FoodState})
	}
	return mealDoc{Name: name, Components: components, RecipeServings: req.RecipeServings, Preparation: strings.TrimSpace(req.Preparation)}, nil
}

func mealPB(doc mealDoc) *pb.Meal {
	components := make([]*pb.MealComponent, 0, len(doc.Components))
	for _, item := range doc.Components {
		components = append(components, &pb.MealComponent{Position: uint32(item.Position), FoodType: item.FoodType,
			ProviderFoodId: item.ProviderFoodID, CustomFoodId: item.CustomFoodID,
			Quantity: &pb.FoodQuantity{Amount: item.Quantity.Amount.String(), Unit: item.Quantity.Unit}, FoodState: item.FoodState})
	}
	return &pb.Meal{Id: doc.ID.Hex(), Version: uint64(doc.Version), Name: doc.Name, Components: components,
		RecipeServings: doc.RecipeServings, Preparation: doc.Preparation,
		CreatedAt: doc.CreatedAt.UTC().Format(time.RFC3339Nano), UpdatedAt: doc.UpdatedAt.UTC().Format(time.RFC3339Nano)}
}

func (s *Server) CreateMeal(ctx context.Context, req *pb.UpsertMealRequest) (*pb.Meal, error) {
	userID, err := Principal(ctx)
	if err != nil {
		return nil, err
	}
	if req.Id != "" || req.Version != 0 {
		return nil, invalid("ID and version are not accepted on create")
	}
	doc, err := s.mealInput(ctx, userID, req)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	doc.ID, doc.UserID, doc.Version, doc.CreatedAt, doc.UpdatedAt = primitive.NewObjectID(), userID, 1, now, now
	if _, err := s.DB.Collection("meals").InsertOne(ctx, doc); err != nil {
		return nil, internal()
	}
	return mealPB(doc), nil
}

func (s *Server) UpdateMeal(ctx context.Context, req *pb.UpsertMealRequest) (*pb.Meal, error) {
	userID, err := Principal(ctx)
	if err != nil {
		return nil, err
	}
	id, err := primitive.ObjectIDFromHex(req.Id)
	if err != nil || req.Version == 0 {
		return nil, invalid("ID and version are required")
	}
	input, err := s.mealInput(ctx, userID, req)
	if err != nil {
		return nil, err
	}
	var doc mealDoc
	err = s.DB.Collection("meals").FindOneAndUpdate(ctx, bson.M{"_id": id, "user_id": userID, "version": int64(req.Version), "deleted_at": bson.M{"$exists": false}},
		bson.M{"$set": bson.M{"name": input.Name, "components": input.Components, "recipe_servings": input.RecipeServings, "preparation": input.Preparation, "updated_at": time.Now().UTC()}, "$inc": bson.M{"version": 1}},
		options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&doc)
	if err == mongo.ErrNoDocuments {
		return nil, status.Error(codes.Aborted, "meal version conflict")
	}
	if err != nil {
		return nil, internal()
	}
	return mealPB(doc), nil
}

func (s *Server) GetMeal(ctx context.Context, req *pb.MealRequest) (*pb.Meal, error) {
	userID, err := Principal(ctx)
	if err != nil {
		return nil, err
	}
	id, err := primitive.ObjectIDFromHex(req.Id)
	if err != nil {
		return nil, invalid("invalid meal ID")
	}
	var doc mealDoc
	err = s.DB.Collection("meals").FindOne(ctx, bson.M{"_id": id, "user_id": userID, "deleted_at": bson.M{"$exists": false}}).Decode(&doc)
	if err == mongo.ErrNoDocuments {
		return nil, status.Error(codes.NotFound, "meal not found")
	}
	if err != nil {
		return nil, internal()
	}
	return mealPB(doc), nil
}

func (s *Server) ListMeals(ctx context.Context, req *pb.ListMealsRequest) (*pb.MealPage, error) {
	userID, err := Principal(ctx)
	if err != nil {
		return nil, err
	}
	if req.Limit == 0 || req.Limit > 50 {
		return nil, invalid("invalid page limit")
	}
	filter := bson.M{"user_id": userID, "deleted_at": bson.M{"$exists": false}}
	if req.Cursor != "" {
		id, err := primitive.ObjectIDFromHex(req.Cursor)
		if err != nil {
			return nil, invalid("invalid cursor")
		}
		filter["_id"] = bson.M{"$lt": id}
	}
	rows, err := s.DB.Collection("meals").Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "_id", Value: -1}}).SetLimit(int64(req.Limit)+1))
	if err != nil {
		return nil, internal()
	}
	defer rows.Close(ctx)
	page := &pb.MealPage{Items: []*pb.Meal{}}
	for rows.Next(ctx) {
		var doc mealDoc
		if rows.Decode(&doc) != nil {
			return nil, internal()
		}
		if len(page.Items) == int(req.Limit) {
			page.HasMore = true
			break
		}
		page.Items = append(page.Items, mealPB(doc))
	}
	if rows.Err() != nil {
		return nil, internal()
	}
	if page.HasMore {
		page.NextCursor = page.Items[len(page.Items)-1].Id
	}
	return page, nil
}

func (s *Server) DeleteMeal(ctx context.Context, req *pb.MealRequest) (*pb.FoodEmpty, error) {
	userID, err := Principal(ctx)
	if err != nil {
		return nil, err
	}
	id, err := primitive.ObjectIDFromHex(req.Id)
	if err != nil {
		return nil, invalid("invalid meal ID")
	}
	result, err := s.DB.Collection("meals").UpdateOne(ctx, bson.M{"_id": id, "user_id": userID, "deleted_at": bson.M{"$exists": false}},
		bson.M{"$set": bson.M{"deleted_at": time.Now().UTC()}})
	if err != nil {
		return nil, internal()
	}
	if result.MatchedCount == 0 {
		return nil, status.Error(codes.NotFound, "meal not found")
	}
	return &pb.FoodEmpty{}, nil
}
