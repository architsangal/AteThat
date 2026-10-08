package server

import (
	"context"
	"strconv"
	"strings"
	"time"

	pb "github.com/architsangal/AteThat/services/core/internal/gen/atethat/v1"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func favouritePB(doc foodFavouriteDoc) *pb.FoodFavourite {
	return &pb.FoodFavourite{FoodKey: doc.FoodKey, FoodType: doc.FoodType,
		ProviderFoodId: doc.ProviderFoodID, CustomFoodId: doc.CustomFoodID, Name: doc.Name,
		CreatedAt: doc.CreatedAt.UTC().Format(time.RFC3339Nano)}
}

func (s *Server) AddFavourite(ctx context.Context, req *pb.FavouriteRequest) (*pb.FoodFavourite, error) {
	userID, err := Principal(ctx)
	if err != nil {
		return nil, err
	}
	doc := foodFavouriteDoc{ID: primitive.NewObjectID(), UserID: userID, FoodType: req.FoodType,
		ProviderFoodID: req.ProviderFoodId, CustomFoodID: req.CustomFoodId, CreatedAt: time.Now().UTC()}
	if req.FoodType == "USDA" {
		id, err := strconv.ParseInt(req.ProviderFoodId, 10, 64)
		if err != nil || id < 1 || req.CustomFoodId != "" {
			return nil, invalid("invalid USDA reference")
		}
		food, err := s.GetUSDAFood(ctx, &pb.GetUSDAFoodRequest{ProviderFoodId: req.ProviderFoodId})
		if err != nil {
			return nil, err
		}
		doc.FoodKey = "USDA:" + req.ProviderFoodId
		doc.Name = food.Name
	} else if req.FoodType == "CUSTOM" {
		if req.ProviderFoodId != "" {
			return nil, invalid("invalid custom food reference")
		}
		food, err := s.GetCustomFood(ctx, &pb.CustomFoodRequest{Id: req.CustomFoodId})
		if err != nil {
			return nil, err
		}
		doc.FoodKey = "CUSTOM:" + food.Id
		doc.Name = food.Name
	} else {
		return nil, invalid("invalid favourite type")
	}
	_, err = s.DB.Collection("food_favourites").UpdateOne(ctx, bson.M{"user_id": userID, "food_key": doc.FoodKey},
		bson.M{"$setOnInsert": doc}, options.Update().SetUpsert(true))
	if err != nil {
		return nil, internal()
	}
	var stored foodFavouriteDoc
	if s.DB.Collection("food_favourites").FindOne(ctx, bson.M{"user_id": userID, "food_key": doc.FoodKey}).Decode(&stored) != nil {
		return nil, internal()
	}
	return favouritePB(stored), nil
}

func (s *Server) DeleteFavourite(ctx context.Context, req *pb.FavouriteRequest) (*pb.FoodEmpty, error) {
	userID, err := Principal(ctx)
	if err != nil {
		return nil, err
	}
	if len(req.FoodKey) > 100 || !(strings.HasPrefix(req.FoodKey, "USDA:") || strings.HasPrefix(req.FoodKey, "CUSTOM:")) {
		return nil, invalid("invalid food key")
	}
	if _, err := s.DB.Collection("food_favourites").DeleteOne(ctx, bson.M{"user_id": userID, "food_key": req.FoodKey}); err != nil {
		return nil, internal()
	}
	return &pb.FoodEmpty{}, nil
}

func (s *Server) ListFavourites(ctx context.Context, req *pb.ListFavouritesRequest) (*pb.FavouritePage, error) {
	userID, err := Principal(ctx)
	if err != nil {
		return nil, err
	}
	if req.Limit == 0 || req.Limit > 50 {
		return nil, invalid("invalid page limit")
	}
	filter := bson.M{"user_id": userID}
	if req.Cursor != "" {
		id, err := primitive.ObjectIDFromHex(req.Cursor)
		if err != nil {
			return nil, invalid("invalid cursor")
		}
		filter["_id"] = bson.M{"$lt": id}
	}
	rows, err := s.DB.Collection("food_favourites").Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "_id", Value: -1}}).SetLimit(int64(req.Limit)+1))
	if err != nil {
		return nil, internal()
	}
	defer rows.Close(ctx)
	page := &pb.FavouritePage{Items: []*pb.FoodFavourite{}}
	var lastID primitive.ObjectID
	for rows.Next(ctx) {
		var doc foodFavouriteDoc
		if rows.Decode(&doc) != nil {
			return nil, internal()
		}
		if len(page.Items) == int(req.Limit) {
			page.HasMore = true
			break
		}
		page.Items = append(page.Items, favouritePB(doc))
		lastID = doc.ID
	}
	if rows.Err() != nil {
		return nil, internal()
	}
	if page.HasMore {
		page.NextCursor = lastID.Hex()
	}
	return page, nil
}
