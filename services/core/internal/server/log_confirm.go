package server

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
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

func parsePreviewToken(raw string) (primitive.ObjectID, string, error) {
	parts := strings.Split(raw, ".")
	if len(parts) != 2 {
		return primitive.NilObjectID, "", invalid("invalid preview token")
	}
	id, err := primitive.ObjectIDFromHex(parts[0])
	if err != nil {
		return primitive.NilObjectID, "", invalid("invalid preview token")
	}
	secret, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || len(secret) != 32 {
		return primitive.NilObjectID, "", invalid("invalid preview token")
	}
	hash := sha256.Sum256(secret)
	return id, hex.EncodeToString(hash[:]), nil
}

func (s *Server) Confirm(ctx context.Context, req *pb.ConfirmFoodGroupRequest) (*pb.FoodLogGroup, error) {
	userID, err := Principal(ctx)
	if err != nil {
		return nil, err
	}
	if len(req.IdempotencyKey) < 8 || len(req.IdempotencyKey) > 128 {
		return nil, invalid("Idempotency-Key must be 8–128 characters")
	}
	previewID, hash, err := parsePreviewToken(req.PreviewToken)
	if err != nil {
		return nil, err
	}
	groupCollection := s.DB.Collection("food_log_groups")
	var existing foodLogGroupDoc
	err = groupCollection.FindOne(ctx, bson.M{"user_id": userID, "idempotency_key": req.IdempotencyKey}).Decode(&existing)
	if err == nil {
		if existing.PreviewID != previewID || existing.PreviewSecretHash != hash {
			return nil, status.Error(codes.AlreadyExists, "idempotency key reused")
		}
		result, err := s.loadGroup(ctx, existing)
		if err == nil {
			result.WasReplayed = true
		}
		return result, err
	}
	if err != mongo.ErrNoDocuments {
		return nil, internal()
	}
	now := time.Now().UTC()
	session, err := s.Client.StartSession()
	if err != nil {
		return nil, internal()
	}
	defer session.EndSession(ctx)
	var group foodLogGroupDoc
	_, err = session.WithTransaction(ctx, func(sc mongo.SessionContext) (interface{}, error) {
		var preview foodPreviewDoc
		err := s.DB.Collection("food_log_previews").FindOne(sc,
			bson.M{"_id": previewID, "user_id": userID, "preview_secret_hash": hash,
				"status": "OPEN", "expires_at": bson.M{"$gt": now}}).Decode(&preview)
		if err != nil {
			return nil, err
		}
		groupID := primitive.NewObjectID()
		group = foodLogGroupDoc{ID: groupID, UserID: userID, RootGroupID: groupID, Revision: 1,
			Status: "CONFIRMED", IdempotencyKey: req.IdempotencyKey, PreviewID: previewID,
			SourceType: preview.SourceType, SourceMealID: preview.SourceMealID,
			PreviewSecretHash: hash, LoggedAt: preview.LoggedAt, TrackingDay: preview.TrackingDay,
			TimezoneAtLogging: preview.TimezoneAtLogging, DailyResetTimeAtLog: preview.DailyResetTimeAtLog,
			TrackingDayRuleVersion: preview.TrackingDayRuleVersion, MealCategory: preview.MealCategory,
			Label: preview.Label, ItemCount: len(preview.Items), Aggregate: preview.Aggregate,
			CalculationVersion: preview.CalculationVersion, ConfirmedAt: now}
		if preview.SourceGroupID != nil {
			var previous foodLogGroupDoc
			if err := groupCollection.FindOne(sc, bson.M{"_id": *preview.SourceGroupID, "user_id": userID, "status": "CONFIRMED"}).Decode(&previous); err != nil {
				return nil, err
			}
			group.RootGroupID = previous.RootGroupID
			group.Revision = previous.Revision + 1
			group.SupersedesGroupID = &previous.ID
			group.CorrectionReason = preview.CorrectionReason
			updated, err := groupCollection.UpdateOne(sc, bson.M{"_id": previous.ID, "user_id": userID, "status": "CONFIRMED"},
				bson.M{"$set": bson.M{"status": "SUPERSEDED"}})
			if err != nil {
				return nil, err
			}
			if updated.MatchedCount != 1 {
				return nil, errors.New("source group changed")
			}
		}
		if _, err := groupCollection.InsertOne(sc, group); err != nil {
			return nil, err
		}
		rows := make([]interface{}, 0, len(preview.Items))
		for _, item := range preview.Items {
			item.ID = primitive.NewObjectID()
			item.UserID, item.GroupID = userID, groupID
			item.LoggedAt, item.TrackingDay, item.ConfirmedAt = preview.LoggedAt, preview.TrackingDay, now
			rows = append(rows, item)
		}
		if _, err := s.DB.Collection("food_logs").InsertMany(sc, rows); err != nil {
			return nil, err
		}
		updated, err := s.DB.Collection("food_log_previews").UpdateOne(sc,
			bson.M{"_id": previewID, "user_id": userID, "status": "OPEN", "expires_at": bson.M{"$gt": now}},
			bson.M{"$set": bson.M{"status": "CONSUMED", "consumed_at": now}})
		if err != nil {
			return nil, err
		}
		if updated.MatchedCount != 1 {
			return nil, errors.New("preview already consumed")
		}
		return nil, nil
	})
	if err != nil {
		// A concurrent retry can win the unique idempotency index. Read it
		// again and return its original result only when the token matches.
		if groupCollection.FindOne(ctx, bson.M{"user_id": userID, "idempotency_key": req.IdempotencyKey}).Decode(&existing) == nil {
			if existing.PreviewID == previewID && existing.PreviewSecretHash == hash {
				result, err := s.loadGroup(ctx, existing)
				if err == nil {
					result.WasReplayed = true
				}
				return result, err
			}
			return nil, status.Error(codes.AlreadyExists, "idempotency key reused")
		}
		return nil, status.Error(codes.FailedPrecondition, "preview expired or consumed")
	}
	return s.loadGroup(ctx, group)
}

func groupPB(group foodLogGroupDoc, items []foodLogItemDoc) *pb.FoodLogGroup {
	result := &pb.FoodLogGroup{Id: group.ID.Hex(), RootGroupId: group.RootGroupID.Hex(),
		Revision: uint64(group.Revision), Status: group.Status,
		LoggedAt: group.LoggedAt.UTC().Format(time.RFC3339Nano), TrackingDay: group.TrackingDay,
		TimezoneAtLogging: group.TimezoneAtLogging, DailyResetTimeAtLogging: group.DailyResetTimeAtLog,
		MealCategory: group.MealCategory, Label: group.Label, ItemCount: uint32(group.ItemCount),
		AggregateFinalNutrition: nutrientPB(group.Aggregate), CalculationVersion: group.CalculationVersion,
		ConfirmedAt: group.ConfirmedAt.UTC().Format(time.RFC3339Nano), Items: []*pb.FoodLogItemSnapshot{}}
	for _, item := range items {
		result.Items = append(result.Items, itemPB(item))
	}
	if group.SupersedesGroupID != nil {
		result.SupersedesGroupId = group.SupersedesGroupID.Hex()
	}
	result.CorrectionReason = group.CorrectionReason
	return result
}

func (s *Server) loadGroup(ctx context.Context, group foodLogGroupDoc) (*pb.FoodLogGroup, error) {
	rows, err := s.DB.Collection("food_logs").Find(ctx, bson.M{"user_id": group.UserID, "food_log_group_id": group.ID},
		options.Find().SetSort(bson.D{{Key: "item_position", Value: 1}}))
	if err != nil {
		return nil, internal()
	}
	defer rows.Close(ctx)
	items := []foodLogItemDoc{}
	if rows.All(ctx, &items) != nil {
		return nil, internal()
	}
	return groupPB(group, items), nil
}
