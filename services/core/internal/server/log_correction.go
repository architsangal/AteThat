package server

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
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

func (s *Server) ownedLogItem(ctx context.Context, rawID string) (foodLogItemDoc, error) {
	userID, err := Principal(ctx)
	if err != nil {
		return foodLogItemDoc{}, err
	}
	id, err := primitive.ObjectIDFromHex(rawID)
	if err != nil {
		return foodLogItemDoc{}, invalid("invalid food log ID")
	}
	var doc foodLogItemDoc
	err = s.DB.Collection("food_logs").FindOne(ctx, bson.M{"_id": id, "user_id": userID}).Decode(&doc)
	if err == mongo.ErrNoDocuments {
		return foodLogItemDoc{}, status.Error(codes.NotFound, "food log not found")
	}
	if err != nil {
		return foodLogItemDoc{}, internal()
	}
	return doc, nil
}

func (s *Server) GetItemLog(ctx context.Context, req *pb.FoodLogItemRequest) (*pb.FoodLogItemSnapshot, error) {
	doc, err := s.ownedLogItem(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return itemPB(doc), nil
}

func (s *Server) CorrectItem(ctx context.Context, req *pb.CorrectFoodLogItemRequest) (*pb.FoodGroupPreview, error) {
	userID, err := Principal(ctx)
	if err != nil {
		return nil, err
	}
	item, err := s.ownedLogItem(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if req.Quantity == nil || req.Quantity.Unit != item.Quantity.Unit || len(strings.TrimSpace(req.Reason)) < 3 || len(req.Reason) > 200 {
		return nil, invalid("quantity unit must match and correction reason is required")
	}
	amount, ok := positiveDecimal(req.Quantity.Amount)
	if !ok {
		return nil, invalid("invalid corrected quantity")
	}
	var group foodLogGroupDoc
	err = s.DB.Collection("food_log_groups").FindOne(ctx,
		bson.M{"_id": item.GroupID, "user_id": userID, "status": "CONFIRMED"}).Decode(&group)
	if err == mongo.ErrNoDocuments {
		return nil, status.Error(codes.FailedPrecondition, "confirmed group not found")
	}
	if err != nil {
		return nil, internal()
	}
	if group.ConfirmedAt.Before(time.Now().UTC().AddDate(0, 0, -90)) {
		return nil, status.Error(codes.FailedPrecondition, "food detail is beyond retention window")
	}
	rows, err := s.DB.Collection("food_logs").Find(ctx,
		bson.M{"user_id": userID, "food_log_group_id": group.ID}, options.Find().SetSort(bson.D{{Key: "item_position", Value: 1}}))
	if err != nil {
		return nil, internal()
	}
	defer rows.Close(ctx)
	items := []foodLogItemDoc{}
	if rows.All(ctx, &items) != nil || len(items) != group.ItemCount {
		return nil, internal()
	}
	for index := range items {
		if items[index].ID != item.ID {
			continue
		}
		old := items[index].Quantity.Amount.String()
		provider, err := scaleNutrition(items[index].ProviderNutrition, amount.String(), old)
		if err != nil {
			return nil, internal()
		}
		overrides, err := scaleNutrition(items[index].UserOverrideNutrition, amount.String(), old)
		if err != nil {
			return nil, internal()
		}
		final := make(map[string]foodNutrientDoc, len(provider)+len(overrides))
		for key, value := range provider {
			final[key] = value
		}
		for key, value := range overrides {
			final[key] = value
		}
		items[index].Quantity.Amount = amount
		items[index].ProviderNutrition = provider
		items[index].UserOverrideNutrition = overrides
		items[index].FinalNutrition = final
	}
	aggregate, err := aggregateNutrition(items)
	if err != nil {
		return nil, internal()
	}
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, internal()
	}
	hash := sha256.Sum256(secret)
	now := time.Now().UTC()
	preview := foodPreviewDoc{ID: primitive.NewObjectID(), UserID: userID, SecretHash: hex.EncodeToString(hash[:]),
		Status: "OPEN", SourceType: group.SourceType, SourceMealID: group.SourceMealID, SourceGroupID: &group.ID,
		CorrectionReason: strings.TrimSpace(req.Reason), LoggedAt: group.LoggedAt, TrackingDay: group.TrackingDay,
		TimezoneAtLogging: group.TimezoneAtLogging, DailyResetTimeAtLog: group.DailyResetTimeAtLog,
		TrackingDayRuleVersion: group.TrackingDayRuleVersion, MealCategory: group.MealCategory, Label: group.Label,
		Items: items, Aggregate: aggregate, CalculationVersion: foodCalculationVersion,
		ExpiresAt: now.Add(15 * time.Minute), CreatedAt: now}
	if _, err := s.DB.Collection("food_log_previews").InsertOne(ctx, preview); err != nil {
		return nil, internal()
	}
	return &pb.FoodGroupPreview{PreviewToken: preview.ID.Hex() + "." + base64.RawURLEncoding.EncodeToString(secret),
		ExpiresAt: preview.ExpiresAt.Format(time.RFC3339Nano), Group: previewGroupPB(preview)}, nil
}

func (s *Server) VoidItem(ctx context.Context, req *pb.FoodLogItemRequest) (*pb.FoodEmpty, error) {
	item, err := s.ownedLogItem(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return s.VoidGroup(ctx, &pb.FoodGroupRequest{Id: item.GroupID.Hex()})
}
