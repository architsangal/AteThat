package server

import (
	"context"
	pb "github.com/architsangal/AteThat/services/core/internal/gen/atethat/v1"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (s *Server) ScheduleHistory(ctx context.Context, r *pb.ScheduleHistoryRequest) (*pb.ScheduleHistoryPage, error) {
	user, err := Principal(ctx)
	if err != nil {
		return nil, err
	}
	if _, err = s.GetItem(ctx, &pb.HealthItemRequest{Id: r.ItemId}); err != nil {
		return nil, err
	}
	if r.Limit < 1 || r.Limit > 50 {
		return nil, invalid("history limit must be 1–50")
	}
	id, _ := primitive.ObjectIDFromHex(r.ItemId)
	filter := bson.M{"user_id": user, "health_item_id": id}
	if r.BeforeVersion > 0 {
		filter["schedule_version"] = bson.M{"$lt": int64(r.BeforeVersion)}
	}
	rows, err := s.DB.Collection("health_item_schedule_versions").Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "schedule_version", Value: -1}}).SetLimit(int64(r.Limit)+1))
	if err != nil {
		return nil, internal()
	}
	defer rows.Close(ctx)
	page := &pb.ScheduleHistoryPage{Items: []*pb.HealthSchedule{}}
	for rows.Next(ctx) {
		var d healthScheduleDoc
		if rows.Decode(&d) != nil {
			return nil, internal()
		}
		if len(page.Items) == int(r.Limit) {
			page.NextBeforeVersion = page.Items[len(page.Items)-1].Version
			break
		}
		page.Items = append(page.Items, healthSchedulePB(d))
	}
	if rows.Err() != nil {
		return nil, internal()
	}
	return page, nil
}
