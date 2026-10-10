package server

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"time"

	pb "github.com/architsangal/AteThat/services/core/internal/gen/atethat/v1"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type usedFoodRow struct {
	FoodKey        string    `bson:"_id"`
	FoodType       string    `bson:"food_type"`
	ProviderFoodID string    `bson:"provider_food_id"`
	CustomFoodID   string    `bson:"custom_food_id"`
	Name           string    `bson:"name"`
	Count          int64     `bson:"confirmed_log_count"`
	LastLoggedAt   time.Time `bson:"last_logged_at"`
}

type usedFoodCursor struct {
	Key    string `json:"k"`
	At     string `json:"t"`
	Count  int64  `json:"c"`
	Filter string `json:"f"`
}

func (s *Server) ListUsedFoods(ctx context.Context, req *pb.ListUsedFoodsRequest) (*pb.UsedFoodPage, error) {
	userID, err := Principal(ctx)
	if err != nil {
		return nil, err
	}
	if req.Sort != "RECENT" && req.Sort != "FREQUENT" {
		return nil, invalid("invalid used-food sort")
	}
	if req.Limit == 0 || req.Limit > 50 {
		return nil, invalid("invalid page limit")
	}
	now := time.Now().UTC()
	from, to := now.AddDate(0, 0, -90), now.Add(5*time.Minute)
	if req.From != "" {
		parsed, err := time.Parse(time.RFC3339Nano, req.From)
		if err != nil {
			return nil, invalid("invalid from")
		}
		from = parsed.UTC()
	}
	if req.To != "" {
		parsed, err := time.Parse(time.RFC3339Nano, req.To)
		if err != nil {
			return nil, invalid("invalid to")
		}
		to = parsed.UTC()
	}
	if !from.Before(to) || to.Sub(from) > 91*24*time.Hour {
		return nil, invalid("used-food range must be at most 91 days")
	}
	filterHash := sha256.Sum256([]byte(req.Sort + "|" + req.From + "|" + req.To))
	filterKey := hex.EncodeToString(filterHash[:])[:24]
	var cursor usedFoodCursor
	if req.Cursor != "" {
		decoded, err := base64.RawURLEncoding.DecodeString(req.Cursor)
		if err != nil || len(decoded) > 512 || json.Unmarshal(decoded, &cursor) != nil || cursor.Filter != filterKey || cursor.Key == "" {
			return nil, invalid("invalid used-food cursor")
		}
	}
	foodID := bson.M{"$cond": bson.A{bson.M{"$eq": bson.A{"$food_source.type", "USDA"}},
		"$food_source.provider_food_id", "$food_source.custom_food_id"}}
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"user_id": userID, "logged_at_utc": bson.M{"$gte": from, "$lt": to}}}},
		{{Key: "$lookup", Value: bson.M{"from": "food_log_groups", "localField": "food_log_group_id", "foreignField": "_id", "as": "group"}}},
		{{Key: "$unwind", Value: "$group"}},
		{{Key: "$match", Value: bson.M{"group.user_id": userID, "group.status": "CONFIRMED"}}},
		{{Key: "$addFields", Value: bson.M{"food_key": bson.M{"$concat": bson.A{"$food_source.type", ":", foodID}}}}},
		{{Key: "$sort", Value: bson.D{{Key: "logged_at_utc", Value: -1}, {Key: "_id", Value: -1}}}},
		{{Key: "$group", Value: bson.M{"_id": "$food_key", "food_type": bson.M{"$first": "$food_source.type"},
			"provider_food_id":    bson.M{"$first": "$food_source.provider_food_id"},
			"custom_food_id":      bson.M{"$first": "$food_source.custom_food_id"},
			"name":                bson.M{"$first": "$food_source.name"},
			"confirmed_log_count": bson.M{"$sum": 1}, "last_logged_at": bson.M{"$max": "$logged_at_utc"}}}},
	}
	if req.Cursor != "" {
		if req.Sort == "RECENT" {
			at, err := time.Parse(time.RFC3339Nano, cursor.At)
			if err != nil {
				return nil, invalid("invalid cursor time")
			}
			pipeline = append(pipeline, bson.D{{Key: "$match", Value: bson.M{"$or": bson.A{
				bson.M{"last_logged_at": bson.M{"$lt": at}},
				bson.M{"last_logged_at": at, "_id": bson.M{"$gt": cursor.Key}}}}}})
		} else {
			pipeline = append(pipeline, bson.D{{Key: "$match", Value: bson.M{"$or": bson.A{
				bson.M{"confirmed_log_count": bson.M{"$lt": cursor.Count}},
				bson.M{"confirmed_log_count": cursor.Count, "_id": bson.M{"$gt": cursor.Key}}}}}})
		}
	}
	if req.Sort == "RECENT" {
		pipeline = append(pipeline, bson.D{{Key: "$sort", Value: bson.D{{Key: "last_logged_at", Value: -1}, {Key: "_id", Value: 1}}}})
	} else {
		pipeline = append(pipeline, bson.D{{Key: "$sort", Value: bson.D{{Key: "confirmed_log_count", Value: -1}, {Key: "_id", Value: 1}}}})
	}
	pipeline = append(pipeline, bson.D{{Key: "$limit", Value: int64(req.Limit) + 1}})
	rows, err := s.DB.Collection("food_logs").Aggregate(ctx, pipeline)
	if err != nil {
		return nil, internal()
	}
	defer rows.Close(ctx)
	result := &pb.UsedFoodPage{Items: []*pb.UsedFood{}}
	var last usedFoodRow
	for rows.Next(ctx) {
		var row usedFoodRow
		if rows.Decode(&row) != nil {
			return nil, internal()
		}
		if len(result.Items) == int(req.Limit) {
			result.HasMore = true
			break
		}
		result.Items = append(result.Items, &pb.UsedFood{FoodKey: row.FoodKey, FoodType: row.FoodType,
			ProviderFoodId: row.ProviderFoodID, CustomFoodId: row.CustomFoodID, Name: row.Name,
			ConfirmedLogCount: uint64(row.Count), LastLoggedAt: row.LastLoggedAt.UTC().Format(time.RFC3339Nano)})
		last = row
	}
	if rows.Err() != nil {
		return nil, internal()
	}
	if result.HasMore {
		encoded, _ := json.Marshal(usedFoodCursor{Key: last.FoodKey, At: last.LastLoggedAt.UTC().Format(time.RFC3339Nano),
			Count: last.Count, Filter: filterKey})
		result.NextCursor = base64.RawURLEncoding.EncodeToString(encoded)
	}
	return result, nil
}
