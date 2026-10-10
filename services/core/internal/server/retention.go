package server

import (
	"context"
	"errors"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var skipCompaction = errors.New("day not ready for compaction")

// RunRetentionLoop compacts only fully aged Tracking Days. Summary creation and
// detail removal are one MongoDB transaction; a failed run retains all detail.
func (s *Server) RunRetentionLoop(ctx context.Context) {
	run := func() {
		iteration, cancel := context.WithTimeout(ctx, 10*time.Minute)
		defer cancel()
		if err := s.RunRetentionOnce(iteration); err != nil && ctx.Err() == nil {
			log.Printf("food retention pass failed: %v", err)
		}
	}
	run()
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			run()
		}
	}
}

func (s *Server) RunRetentionOnce(ctx context.Context) error {
	cutoff := time.Now().UTC().AddDate(0, 0, -90)
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"confirmed_at": bson.M{"$lt": cutoff}}}},
		{{Key: "$group", Value: bson.M{"_id": bson.M{"user_id": "$user_id", "tracking_day": "$tracking_day"},
			"latest_confirmation": bson.M{"$max": "$confirmed_at"}}}},
		{{Key: "$match", Value: bson.M{"latest_confirmation": bson.M{"$lt": cutoff}}}},
		{{Key: "$limit", Value: int64(100)}},
	}
	rows, err := s.DB.Collection("food_log_groups").Aggregate(ctx, pipeline)
	if err != nil {
		return err
	}
	defer rows.Close(ctx)
	for rows.Next(ctx) {
		var candidate struct {
			ID struct {
				UserID      primitive.ObjectID `bson:"user_id"`
				TrackingDay string             `bson:"tracking_day"`
			} `bson:"_id"`
		}
		if err := rows.Decode(&candidate); err != nil {
			return err
		}
		if err := s.compactDay(ctx, candidate.ID.UserID, candidate.ID.TrackingDay, cutoff); err != nil && !errors.Is(err, skipCompaction) {
			return err
		}
	}
	return rows.Err()
}

func (s *Server) compactDay(ctx context.Context, userID primitive.ObjectID, day string, cutoff time.Time) error {
	session, err := s.Client.StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)
	_, err = session.WithTransaction(ctx, func(sc mongo.SessionContext) (interface{}, error) {
		filter := bson.M{"user_id": userID, "tracking_day": day}
		if err := s.DB.Collection("daily_nutrition_summaries").FindOne(sc, filter).Err(); err == nil {
			return nil, skipCompaction
		} else if err != mongo.ErrNoDocuments {
			return nil, err
		}
		if err := s.DB.Collection("food_log_previews").FindOne(sc,
			bson.M{"user_id": userID, "tracking_day": day, "status": "OPEN", "expires_at": bson.M{"$gt": time.Now().UTC()}}).Err(); err == nil {
			return nil, skipCompaction
		} else if err != mongo.ErrNoDocuments {
			return nil, err
		}
		rows, err := s.DB.Collection("food_log_groups").Find(sc, filter)
		if err != nil {
			return nil, err
		}
		defer rows.Close(sc)
		ids := []primitive.ObjectID{}
		confirmed := []foodLogItemDoc{}
		groupCount, itemCount := 0, 0
		var earliest, latest time.Time
		var timezone, reset string
		for rows.Next(sc) {
			var group foodLogGroupDoc
			if err := rows.Decode(&group); err != nil {
				return nil, err
			}
			if !group.ConfirmedAt.Before(cutoff) {
				return nil, skipCompaction
			}
			ids = append(ids, group.ID)
			if timezone == "" {
				timezone, reset = group.TimezoneAtLogging, group.DailyResetTimeAtLog
			}
			if group.Status == "CONFIRMED" {
				confirmed = append(confirmed, foodLogItemDoc{FinalNutrition: group.Aggregate})
				if earliest.IsZero() || group.LoggedAt.Before(earliest) {
					earliest = group.LoggedAt
				}
				if latest.IsZero() || group.LoggedAt.After(latest) {
					latest = group.LoggedAt
				}
				groupCount++
				itemCount += group.ItemCount
			}
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
		if err := rows.Close(sc); err != nil {
			return nil, err
		}
		if len(ids) == 0 {
			return nil, skipCompaction
		}
		aggregate := map[string]foodNutrientDoc{}
		if len(confirmed) > 0 {
			aggregate, err = aggregateNutrition(confirmed)
			if err != nil {
				return nil, err
			}
		}
		now := time.Now().UTC()
		summary := dailyNutritionSummaryDoc{ID: primitive.NewObjectID(), UserID: userID, TrackingDay: day,
			TimezoneAtSummary: timezone, DailyResetTime: reset, Aggregate: aggregate,
			ConfirmedGroupCount: groupCount, ConfirmedItemCount: itemCount,
			EarliestLoggedAtUTC: earliest, LatestLoggedAtUTC: latest,
			SchemaVersion:     1,
			DetailCompactedAt: now, SourceDetailThrough: cutoff, CreatedAt: now}
		if _, err := s.DB.Collection("daily_nutrition_summaries").InsertOne(sc, summary); err != nil {
			return nil, err
		}
		if _, err := s.DB.Collection("food_logs").DeleteMany(sc,
			bson.M{"user_id": userID, "food_log_group_id": bson.M{"$in": ids}}); err != nil {
			return nil, err
		}
		if _, err := s.DB.Collection("food_log_groups").DeleteMany(sc,
			bson.M{"user_id": userID, "tracking_day": day, "_id": bson.M{"$in": ids}}); err != nil {
			return nil, err
		}
		return nil, nil
	})
	return err
}
