package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"time"

	pb "github.com/architsangal/AteThat/services/core/internal/gen/atethat/v1"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type healthHistoryCursor struct {
	At   string `json:"t"`
	ID   string `json:"i"`
	From string `json:"f"`
	To   string `json:"e"`
}

func (s *Server) HealthActionHistory(ctx context.Context, req *pb.HealthHistoryRequest) (*pb.HealthHistoryPage, error) {
	userID, err := Principal(ctx)
	if err != nil {
		return nil, err
	}
	from, errA := time.Parse(time.RFC3339Nano, req.From)
	to, errB := time.Parse(time.RFC3339Nano, req.To)
	if errA != nil || errB != nil || !from.Before(to) || to.Sub(from) > 366*24*time.Hour || req.Limit == 0 || req.Limit > 100 {
		return nil, invalid("health history range must be at most 366 days and limit 1–100")
	}
	filter := bson.M{"user_id": userID, "scheduled_at_utc": bson.M{"$gte": from.UTC(), "$lt": to.UTC()}}
	if req.Cursor != "" {
		decoded, err := base64.RawURLEncoding.DecodeString(req.Cursor)
		if err != nil || len(decoded) > 512 {
			return nil, invalid("invalid cursor")
		}
		var cursor healthHistoryCursor
		if json.Unmarshal(decoded, &cursor) != nil || cursor.From != req.From || cursor.To != req.To {
			return nil, invalid("invalid cursor")
		}
		at, errA := time.Parse(time.RFC3339Nano, cursor.At)
		id, errB := primitive.ObjectIDFromHex(cursor.ID)
		if errA != nil || errB != nil {
			return nil, invalid("invalid cursor")
		}
		filter["$or"] = bson.A{bson.M{"scheduled_at_utc": bson.M{"$lt": at}},
			bson.M{"scheduled_at_utc": at, "_id": bson.M{"$lt": id}}}
	}
	rows, err := s.DB.Collection("health_item_events").Find(ctx, filter,
		options.Find().SetSort(bson.D{{Key: "scheduled_at_utc", Value: -1}, {Key: "_id", Value: -1}}).SetLimit(int64(req.Limit)+1))
	if err != nil {
		return nil, internal()
	}
	defer rows.Close(ctx)
	page := &pb.HealthHistoryPage{Items: []*pb.HealthOccurrence{}}
	var last healthEventDoc
	for rows.Next(ctx) {
		var event healthEventDoc
		if rows.Decode(&event) != nil {
			return nil, internal()
		}
		if len(page.Items) == int(req.Limit) {
			page.HasMore = true
			break
		}
		var item healthItemDoc
		if s.DB.Collection("health_items").FindOne(ctx, bson.M{"_id": event.HealthItemID, "user_id": userID}).Decode(&item) != nil {
			return nil, internal()
		}
		occurrence := &pb.HealthOccurrence{OccurrenceId: event.OccurrenceID, HealthItemId: event.HealthItemID.Hex(),
			ScheduleVersion: uint64(event.ScheduleVersion), Name: item.Name, Type: item.Type,
			ScheduledAt: event.ScheduledAt.UTC().Format(time.RFC3339Nano), ScheduledLocalDate: event.LocalDate,
			ScheduledLocalTime: event.LocalTime, Timezone: event.Timezone}
		eventOnOccurrence(occurrence, event)
		page.Items = append(page.Items, occurrence)
		last = event
	}
	if rows.Err() != nil {
		return nil, internal()
	}
	if page.HasMore {
		encoded, _ := json.Marshal(healthHistoryCursor{At: last.ScheduledAt.UTC().Format(time.RFC3339Nano), ID: last.ID.Hex(), From: req.From, To: req.To})
		page.NextCursor = base64.RawURLEncoding.EncodeToString(encoded)
	}
	return page, nil
}
