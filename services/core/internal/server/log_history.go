package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"time"

	pb "github.com/architsangal/AteThat/services/core/internal/gen/atethat/v1"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type groupCursor struct {
	LoggedAt string `json:"t"`
	ID       string `json:"i"`
	Day      string `json:"d"`
}

func (s *Server) ListGroups(ctx context.Context, req *pb.ListFoodGroupsRequest) (*pb.FoodGroupPage, error) {
	userID, err := Principal(ctx)
	if err != nil {
		return nil, err
	}
	if req.Limit == 0 || req.Limit > 50 {
		return nil, invalid("invalid page limit")
	}
	day := req.TrackingDay
	if day == "" {
		current, err := s.GetCurrentTrackingDay(ctx, &pb.Empty{})
		if err != nil {
			return nil, err
		}
		day = current.TrackingDay
	}
	if parsed, err := time.Parse("2006-01-02", day); err != nil || parsed.Format("2006-01-02") != day {
		return nil, invalid("invalid tracking day")
	}
	filter := bson.M{"user_id": userID, "tracking_day": day, "status": "CONFIRMED"}
	if req.Cursor != "" {
		decoded, err := base64.RawURLEncoding.DecodeString(req.Cursor)
		if err != nil || len(decoded) > 256 {
			return nil, invalid("invalid cursor")
		}
		var cursor groupCursor
		if json.Unmarshal(decoded, &cursor) != nil || cursor.Day != day {
			return nil, invalid("invalid cursor")
		}
		loggedAt, timeErr := time.Parse(time.RFC3339Nano, cursor.LoggedAt)
		id, idErr := primitive.ObjectIDFromHex(cursor.ID)
		if timeErr != nil || idErr != nil {
			return nil, invalid("invalid cursor")
		}
		filter["$or"] = []bson.M{{"logged_at_utc": bson.M{"$lt": loggedAt}},
			{"logged_at_utc": loggedAt, "_id": bson.M{"$lt": id}}}
	}
	rows, err := s.DB.Collection("food_log_groups").Find(ctx, filter,
		options.Find().SetSort(bson.D{{Key: "logged_at_utc", Value: -1}, {Key: "_id", Value: -1}}).SetLimit(int64(req.Limit)+1))
	if err != nil {
		return nil, internal()
	}
	defer rows.Close(ctx)
	result := &pb.FoodGroupPage{Items: []*pb.FoodLogGroup{}}
	var last foodLogGroupDoc
	for rows.Next(ctx) {
		var doc foodLogGroupDoc
		if rows.Decode(&doc) != nil {
			return nil, internal()
		}
		if len(result.Items) == int(req.Limit) {
			result.HasMore = true
			break
		}
		result.Items = append(result.Items, groupPB(doc, nil))
		last = doc
	}
	if rows.Err() != nil {
		return nil, internal()
	}
	if result.HasMore {
		encoded, _ := json.Marshal(groupCursor{LoggedAt: last.LoggedAt.UTC().Format(time.RFC3339Nano), ID: last.ID.Hex(), Day: day})
		result.NextCursor = base64.RawURLEncoding.EncodeToString(encoded)
	}
	return result, nil
}

func (s *Server) GetGroup(ctx context.Context, req *pb.FoodGroupRequest) (*pb.FoodLogGroup, error) {
	userID, err := Principal(ctx)
	if err != nil {
		return nil, err
	}
	id, err := primitive.ObjectIDFromHex(req.Id)
	if err != nil {
		return nil, invalid("invalid group ID")
	}
	var group foodLogGroupDoc
	err = s.DB.Collection("food_log_groups").FindOne(ctx, bson.M{"_id": id, "user_id": userID}).Decode(&group)
	if err == mongo.ErrNoDocuments {
		return nil, status.Error(codes.NotFound, "food group not found")
	}
	if err != nil {
		return nil, internal()
	}
	return s.loadGroup(ctx, group)
}

func (s *Server) VoidGroup(ctx context.Context, req *pb.FoodGroupRequest) (*pb.FoodEmpty, error) {
	userID, err := Principal(ctx)
	if err != nil {
		return nil, err
	}
	id, err := primitive.ObjectIDFromHex(req.Id)
	if err != nil {
		return nil, invalid("invalid group ID")
	}
	result, err := s.DB.Collection("food_log_groups").UpdateOne(ctx,
		bson.M{"_id": id, "user_id": userID, "status": "CONFIRMED"},
		bson.M{"$set": bson.M{"status": "VOIDED", "voided_at": time.Now().UTC()}})
	if err != nil {
		return nil, internal()
	}
	if result.MatchedCount == 0 {
		return nil, status.Error(codes.NotFound, "confirmed group not found")
	}
	return &pb.FoodEmpty{}, nil
}
