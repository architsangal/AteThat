package server

import (
	"context"
	"math/big"
	"regexp"
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

// Canonical names include VITAMIN_B6 and VITAMIN_B12. Keep MongoDB path
// operators/separators excluded while allowing digits after the first letter.
var nutrientKeyPattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]{0,39}$`)

func positiveDecimal(raw string) (primitive.Decimal128, bool) {
	if len(raw) > 32 || !nonnegativeDecimalPattern.MatchString(raw) {
		return primitive.Decimal128{}, false
	}
	rational, ok := new(big.Rat).SetString(raw)
	if !ok || rational.Sign() <= 0 {
		return primitive.Decimal128{}, false
	}
	value, err := primitive.ParseDecimal128(raw)
	return value, err == nil
}

func customFoodInput(req *pb.UpsertCustomFoodRequest) (customFoodDoc, error) {
	name := strings.TrimSpace(req.Name)
	if len(name) < 2 || len(name) > 120 || len(req.Description) > 500 || req.BaseQuantity == nil {
		return customFoodDoc{}, invalid("invalid custom food name, description, or quantity")
	}
	if req.FoodState != "RAW" && req.FoodState != "COOKED" && req.FoodState != "PREPARED" && req.FoodState != "UNKNOWN" {
		return customFoodDoc{}, invalid("invalid food state")
	}
	amount, ok := positiveDecimal(req.BaseQuantity.Amount)
	if !ok || (req.BaseQuantity.Unit != "g" && req.BaseQuantity.Unit != "ml") {
		return customFoodDoc{}, invalid("base quantity must be positive g or ml")
	}
	if len(req.Nutrients) == 0 || len(req.Nutrients) > 100 {
		return customFoodDoc{}, invalid("at least one nutrient is required")
	}
	nutrients := make(map[string]foodNutrientDoc, len(req.Nutrients))
	for key, input := range req.Nutrients {
		if !nutrientKeyPattern.MatchString(key) || input == nil || len(input.Unit) < 1 || len(input.Unit) > 20 {
			return customFoodDoc{}, invalid("invalid nutrient")
		}
		if input.Status != "KNOWN" && input.Status != "UNKNOWN" {
			return customFoodDoc{}, invalid("invalid nutrient status")
		}
		entry := foodNutrientDoc{Status: input.Status, Unit: input.Unit}
		if input.Status == "KNOWN" {
			if len(input.Amount) > 32 || !nonnegativeDecimalPattern.MatchString(input.Amount) {
				return customFoodDoc{}, invalid("invalid nutrient amount")
			}
			value, err := primitive.ParseDecimal128(input.Amount)
			if err != nil {
				return customFoodDoc{}, invalid("invalid nutrient amount")
			}
			entry.Amount = &value
		} else if input.Amount != "" {
			return customFoodDoc{}, invalid("unknown nutrient must not have an amount")
		}
		nutrients[key] = entry
	}
	return customFoodDoc{Name: name, Description: strings.TrimSpace(req.Description), FoodState: req.FoodState,
		BaseQuantity: quantityDoc{Amount: amount, Unit: req.BaseQuantity.Unit}, Nutrients: nutrients}, nil
}

func customFoodPB(doc customFoodDoc) *pb.CustomFood {
	nutrients := make(map[string]*pb.FoodNutrient, len(doc.Nutrients))
	for key, value := range doc.Nutrients {
		entry := &pb.FoodNutrient{Status: value.Status, Unit: value.Unit}
		if value.Amount != nil {
			entry.Amount = value.Amount.String()
		}
		nutrients[key] = entry
	}
	return &pb.CustomFood{Id: doc.ID.Hex(), Version: uint64(doc.Version), Name: doc.Name,
		Description: doc.Description, FoodState: doc.FoodState, BaseQuantity: &pb.FoodQuantity{
			Amount: doc.BaseQuantity.Amount.String(), Unit: doc.BaseQuantity.Unit}, Nutrients: nutrients,
		CreatedAt: doc.CreatedAt.UTC().Format(time.RFC3339Nano), UpdatedAt: doc.UpdatedAt.UTC().Format(time.RFC3339Nano)}
}

func (s *Server) CreateCustomFood(ctx context.Context, req *pb.UpsertCustomFoodRequest) (*pb.CustomFood, error) {
	userID, err := Principal(ctx)
	if err != nil {
		return nil, err
	}
	if req.Id != "" || req.Version != 0 {
		return nil, invalid("ID and version are not accepted on create")
	}
	doc, err := customFoodInput(req)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	doc.ID, doc.UserID, doc.Version, doc.CreatedAt, doc.UpdatedAt = primitive.NewObjectID(), userID, 1, now, now
	if _, err := s.DB.Collection("custom_foods").InsertOne(ctx, doc); err != nil {
		return nil, internal()
	}
	return customFoodPB(doc), nil
}

func (s *Server) UpdateCustomFood(ctx context.Context, req *pb.UpsertCustomFoodRequest) (*pb.CustomFood, error) {
	userID, err := Principal(ctx)
	if err != nil {
		return nil, err
	}
	id, err := primitive.ObjectIDFromHex(req.Id)
	if err != nil || req.Version == 0 {
		return nil, invalid("ID and version are required")
	}
	input, err := customFoodInput(req)
	if err != nil {
		return nil, err
	}
	set := bson.M{"name": input.Name, "description": input.Description, "food_state": input.FoodState,
		"base_quantity": input.BaseQuantity, "nutrients": input.Nutrients, "updated_at": time.Now().UTC()}
	var doc customFoodDoc
	err = s.DB.Collection("custom_foods").FindOneAndUpdate(ctx,
		bson.M{"_id": id, "user_id": userID, "version": int64(req.Version), "deleted_at": bson.M{"$exists": false}},
		bson.M{"$set": set, "$inc": bson.M{"version": 1}},
		options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&doc)
	if err == mongo.ErrNoDocuments {
		return nil, status.Error(codes.Aborted, "custom food version conflict or not found")
	}
	if err != nil {
		return nil, internal()
	}
	return customFoodPB(doc), nil
}

func (s *Server) GetCustomFood(ctx context.Context, req *pb.CustomFoodRequest) (*pb.CustomFood, error) {
	userID, err := Principal(ctx)
	if err != nil {
		return nil, err
	}
	id, err := primitive.ObjectIDFromHex(req.Id)
	if err != nil {
		return nil, invalid("invalid custom food ID")
	}
	var doc customFoodDoc
	err = s.DB.Collection("custom_foods").FindOne(ctx,
		bson.M{"_id": id, "user_id": userID, "deleted_at": bson.M{"$exists": false}}).Decode(&doc)
	if err == mongo.ErrNoDocuments {
		return nil, status.Error(codes.NotFound, "custom food not found")
	}
	if err != nil {
		return nil, internal()
	}
	return customFoodPB(doc), nil
}

func (s *Server) ListCustomFoods(ctx context.Context, req *pb.ListCustomFoodsRequest) (*pb.CustomFoodPage, error) {
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
	rows, err := s.DB.Collection("custom_foods").Find(ctx, filter,
		options.Find().SetSort(bson.D{{Key: "_id", Value: -1}}).SetLimit(int64(req.Limit)+1))
	if err != nil {
		return nil, internal()
	}
	defer rows.Close(ctx)
	result := &pb.CustomFoodPage{Items: []*pb.CustomFood{}}
	for rows.Next(ctx) {
		var doc customFoodDoc
		if rows.Decode(&doc) != nil {
			return nil, internal()
		}
		if len(result.Items) == int(req.Limit) {
			result.HasMore = true
			break
		}
		result.Items = append(result.Items, customFoodPB(doc))
	}
	if rows.Err() != nil {
		return nil, internal()
	}
	if result.HasMore {
		result.NextCursor = result.Items[len(result.Items)-1].Id
	}
	return result, nil
}

func (s *Server) DeleteCustomFood(ctx context.Context, req *pb.CustomFoodRequest) (*pb.FoodEmpty, error) {
	userID, err := Principal(ctx)
	if err != nil {
		return nil, err
	}
	id, err := primitive.ObjectIDFromHex(req.Id)
	if err != nil {
		return nil, invalid("invalid custom food ID")
	}
	result, err := s.DB.Collection("custom_foods").UpdateOne(ctx,
		bson.M{"_id": id, "user_id": userID, "deleted_at": bson.M{"$exists": false}},
		bson.M{"$set": bson.M{"deleted_at": time.Now().UTC()}})
	if err != nil {
		return nil, internal()
	}
	if result.MatchedCount == 0 {
		return nil, status.Error(codes.NotFound, "custom food not found")
	}
	return &pb.FoodEmpty{}, nil
}
