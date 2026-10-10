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

func (s *Server) servingInput(ctx context.Context, userID primitive.ObjectID, req *pb.UpsertServingRequest) (servingDoc, error) {
	label := strings.TrimSpace(req.Label)
	if len(label) < 2 || len(label) > 80 || req.Quantity == nil {
		return servingDoc{}, invalid("invalid serving label or quantity")
	}
	amount, ok := positiveDecimal(req.Quantity.Amount)
	if !ok || (req.Quantity.Unit != "g" && req.Quantity.Unit != "ml") {
		return servingDoc{}, invalid("serving quantity must be positive g or ml")
	}
	if req.FoodState != "RAW" && req.FoodState != "COOKED" && req.FoodState != "PREPARED" && req.FoodState != "UNKNOWN" {
		return servingDoc{}, invalid("invalid food state")
	}
	if req.FoodType == "USDA" {
		id, err := strconv.ParseInt(req.ProviderFoodId, 10, 64)
		if err != nil || id < 1 || req.CustomFoodId != "" || req.Quantity.Unit != "g" {
			return servingDoc{}, invalid("invalid USDA reference")
		}
	} else if req.FoodType == "CUSTOM" {
		id, err := primitive.ObjectIDFromHex(req.CustomFoodId)
		if err != nil || req.ProviderFoodId != "" {
			return servingDoc{}, invalid("invalid custom-food reference")
		}
		var custom customFoodDoc
		if err := s.DB.Collection("custom_foods").FindOne(ctx,
			bson.M{"_id": id, "user_id": userID, "deleted_at": bson.M{"$exists": false}}).Decode(&custom); err != nil {
			return servingDoc{}, status.Error(codes.NotFound, "custom food not found")
		}
		if custom.BaseQuantity.Unit != req.Quantity.Unit {
			return servingDoc{}, invalid("serving unit must match custom food base unit")
		}
	} else {
		return servingDoc{}, invalid("invalid food reference type")
	}
	return servingDoc{Label: label, FoodType: req.FoodType, ProviderFoodID: req.ProviderFoodId,
		CustomFoodID: req.CustomFoodId, Quantity: quantityDoc{Amount: amount, Unit: req.Quantity.Unit}, FoodState: req.FoodState}, nil
}

func servingPB(doc servingDoc) *pb.ServingDefinition {
	return &pb.ServingDefinition{Id: doc.ID.Hex(), Version: uint64(doc.Version), Label: doc.Label,
		FoodType: doc.FoodType, ProviderFoodId: doc.ProviderFoodID, CustomFoodId: doc.CustomFoodID,
		Quantity:  &pb.FoodQuantity{Amount: doc.Quantity.Amount.String(), Unit: doc.Quantity.Unit},
		FoodState: doc.FoodState, CreatedAt: doc.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt: doc.UpdatedAt.UTC().Format(time.RFC3339Nano)}
}

func (s *Server) CreateServing(ctx context.Context, req *pb.UpsertServingRequest) (*pb.ServingDefinition, error) {
	userID, err := Principal(ctx)
	if err != nil {
		return nil, err
	}
	if req.Id != "" || req.Version != 0 {
		return nil, invalid("ID and version are not accepted on create")
	}
	doc, err := s.servingInput(ctx, userID, req)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	doc.ID, doc.UserID, doc.SchemaVersion, doc.Version, doc.CreatedAt, doc.UpdatedAt = primitive.NewObjectID(), userID, 1, 1, now, now
	if _, err := s.DB.Collection("serving_definitions").InsertOne(ctx, doc); err != nil {
		return nil, internal()
	}
	return servingPB(doc), nil
}

func (s *Server) UpdateServing(ctx context.Context, req *pb.UpsertServingRequest) (*pb.ServingDefinition, error) {
	userID, err := Principal(ctx)
	if err != nil {
		return nil, err
	}
	id, err := primitive.ObjectIDFromHex(req.Id)
	if err != nil || req.Version == 0 {
		return nil, invalid("ID and version are required")
	}
	input, err := s.servingInput(ctx, userID, req)
	if err != nil {
		return nil, err
	}
	var doc servingDoc
	err = s.DB.Collection("serving_definitions").FindOneAndUpdate(ctx,
		bson.M{"_id": id, "user_id": userID, "version": int64(req.Version), "deleted_at": bson.M{"$exists": false}},
		bson.M{"$set": bson.M{"label": input.Label, "food_type": input.FoodType,
			"provider_food_id": input.ProviderFoodID, "custom_food_id": input.CustomFoodID,
			"quantity": input.Quantity, "food_state": input.FoodState, "updated_at": time.Now().UTC()}, "$inc": bson.M{"version": 1}},
		options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&doc)
	if err == mongo.ErrNoDocuments {
		return nil, status.Error(codes.Aborted, "serving version conflict or not found")
	}
	if err != nil {
		return nil, internal()
	}
	return servingPB(doc), nil
}

func (s *Server) GetServing(ctx context.Context, req *pb.ServingRequest) (*pb.ServingDefinition, error) {
	userID, err := Principal(ctx)
	if err != nil {
		return nil, err
	}
	id, err := primitive.ObjectIDFromHex(req.Id)
	if err != nil {
		return nil, invalid("invalid serving ID")
	}
	var doc servingDoc
	err = s.DB.Collection("serving_definitions").FindOne(ctx,
		bson.M{"_id": id, "user_id": userID, "deleted_at": bson.M{"$exists": false}}).Decode(&doc)
	if err == mongo.ErrNoDocuments {
		return nil, status.Error(codes.NotFound, "serving not found")
	}
	if err != nil {
		return nil, internal()
	}
	return servingPB(doc), nil
}

func (s *Server) ListServings(ctx context.Context, req *pb.ListServingsRequest) (*pb.ServingPage, error) {
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
			return nil, invalid("invalid serving cursor")
		}
		filter["_id"] = bson.M{"$lt": id}
	}
	rows, err := s.DB.Collection("serving_definitions").Find(ctx, filter,
		options.Find().SetSort(bson.D{{Key: "_id", Value: -1}}).SetLimit(int64(req.Limit)+1))
	if err != nil {
		return nil, internal()
	}
	defer rows.Close(ctx)
	page := &pb.ServingPage{Items: []*pb.ServingDefinition{}}
	var last primitive.ObjectID
	for rows.Next(ctx) {
		var doc servingDoc
		if rows.Decode(&doc) != nil {
			return nil, internal()
		}
		if len(page.Items) == int(req.Limit) {
			page.HasMore = true
			break
		}
		page.Items = append(page.Items, servingPB(doc))
		last = doc.ID
	}
	if rows.Err() != nil {
		return nil, internal()
	}
	if page.HasMore {
		page.NextCursor = last.Hex()
	}
	return page, nil
}

func (s *Server) DeleteServing(ctx context.Context, req *pb.ServingRequest) (*pb.FoodEmpty, error) {
	userID, err := Principal(ctx)
	if err != nil {
		return nil, err
	}
	id, err := primitive.ObjectIDFromHex(req.Id)
	if err != nil {
		return nil, invalid("invalid serving ID")
	}
	result, err := s.DB.Collection("serving_definitions").UpdateOne(ctx,
		bson.M{"_id": id, "user_id": userID, "deleted_at": bson.M{"$exists": false}},
		bson.M{"$set": bson.M{"deleted_at": time.Now().UTC()}})
	if err != nil {
		return nil, internal()
	}
	if result.MatchedCount == 0 {
		return nil, status.Error(codes.NotFound, "serving not found")
	}
	return &pb.FoodEmpty{}, nil
}
