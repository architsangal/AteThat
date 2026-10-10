package server

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"
	"time"

	"github.com/architsangal/AteThat/services/core/internal/foods"
	pb "github.com/architsangal/AteThat/services/core/internal/gen/atethat/v1"
	"github.com/architsangal/AteThat/services/core/internal/trackingday"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const foodCalculationVersion = "nutrition-normalizer-v1"

func nutritionDoc(input map[string]*pb.FoodNutrient) (map[string]foodNutrientDoc, error) {
	result := make(map[string]foodNutrientDoc, len(input))
	for key, value := range input {
		if !nutrientKeyPattern.MatchString(key) || value == nil || value.Unit == "" || len(value.Unit) > 20 {
			return nil, invalid("invalid nutrient")
		}
		entry := foodNutrientDoc{Status: value.Status, Unit: value.Unit}
		if value.Status == "KNOWN" {
			if len(value.Amount) > 32 || !nonnegativeDecimalPattern.MatchString(value.Amount) {
				return nil, invalid("invalid nutrient amount")
			}
			amount, err := primitive.ParseDecimal128(value.Amount)
			if err != nil {
				return nil, invalid("invalid nutrient amount")
			}
			entry.Amount = &amount
		} else if value.Status != "UNKNOWN" || value.Amount != "" {
			return nil, invalid("invalid unknown nutrient")
		}
		result[key] = entry
	}
	return result, nil
}

func nutrientPB(input map[string]foodNutrientDoc) map[string]*pb.FoodNutrient {
	result := make(map[string]*pb.FoodNutrient, len(input))
	for key, value := range input {
		entry := &pb.FoodNutrient{Status: value.Status, Unit: value.Unit}
		if value.Amount != nil {
			entry.Amount = value.Amount.String()
		}
		result[key] = entry
	}
	return result
}

func scaleNutrition(input map[string]foodNutrientDoc, quantity, base string) (map[string]foodNutrientDoc, error) {
	result := make(map[string]foodNutrientDoc, len(input))
	for key, value := range input {
		if value.Status != "KNOWN" || value.Amount == nil {
			result[key] = value
			continue
		}
		scaled, err := foods.Scale(value.Amount.String(), quantity, base)
		if err != nil {
			return nil, err
		}
		amount, err := primitive.ParseDecimal128(scaled)
		if err != nil {
			return nil, err
		}
		value.Amount = &amount
		result[key] = value
	}
	return result, nil
}

func aggregateNutrition(items []foodLogItemDoc) (map[string]foodNutrientDoc, error) {
	keys := map[string]struct{}{}
	for _, item := range items {
		for key := range item.FinalNutrition {
			keys[key] = struct{}{}
		}
	}
	result := make(map[string]foodNutrientDoc, len(keys))
	for key := range keys {
		values := []string{}
		unit, known := "", true
		for _, item := range items {
			value, exists := item.FinalNutrition[key]
			if exists && value.Unit != "" {
				if unit != "" && unit != value.Unit {
					known = false
				}
				unit = value.Unit
			}
			if !exists || value.Status != "KNOWN" || value.Amount == nil {
				known = false
				continue
			}
			values = append(values, value.Amount.String())
		}
		if !known {
			result[key] = foodNutrientDoc{Status: "UNKNOWN", Unit: unit}
			continue
		}
		total, err := foods.Sum(values)
		if err != nil {
			return nil, err
		}
		amount, err := primitive.ParseDecimal128(total)
		if err != nil {
			return nil, err
		}
		result[key] = foodNutrientDoc{Status: "KNOWN", Amount: &amount, Unit: unit}
	}
	return result, nil
}

func (s *Server) Preview(ctx context.Context, req *pb.PreviewFoodGroupRequest) (*pb.FoodGroupPreview, error) {
	userID, err := Principal(ctx)
	if err != nil {
		return nil, err
	}
	if len(req.Items) < 1 || len(req.Items) > 50 || len(req.Label) > 160 {
		return nil, invalid("food group requires 1–50 items and a short label")
	}
	if req.MealCategory != "BREAKFAST" && req.MealCategory != "LUNCH" && req.MealCategory != "DINNER" && req.MealCategory != "SNACK" && req.MealCategory != "OTHER" {
		return nil, invalid("invalid meal category")
	}
	loggedAt, err := time.Parse(time.RFC3339Nano, req.LoggedAt)
	if err != nil {
		return nil, invalid("logged_at must include a timezone offset")
	}
	loggedAt = loggedAt.UTC()
	if loggedAt.After(time.Now().Add(5*time.Minute)) || loggedAt.Before(time.Now().AddDate(0, 0, -90)) {
		return nil, invalid("logged_at must be within the last 90 days")
	}
	var profile profileDoc
	var preferences preferencesDoc
	if s.DB.Collection("user_profiles").FindOne(ctx, bson.M{"user_id": userID}).Decode(&profile) != nil ||
		s.DB.Collection("user_preferences").FindOne(ctx, bson.M{"user_id": userID}).Decode(&preferences) != nil {
		return nil, internal()
	}
	day, err := trackingday.Derive(loggedAt, profile.Timezone, preferences.DailyResetTime)
	if err != nil {
		return nil, internal()
	}
	if err := s.DB.Collection("daily_nutrition_summaries").FindOne(ctx,
		bson.M{"user_id": userID, "tracking_day": day.Day}).Err(); err == nil {
		return nil, status.Error(codes.FailedPrecondition, "compacted Tracking Day is read-only")
	} else if err != mongo.ErrNoDocuments {
		return nil, internal()
	}
	items := make([]foodLogItemDoc, 0, len(req.Items))
	for index, input := range req.Items {
		if input == nil || int(input.Position) != index || input.Quantity == nil {
			return nil, invalid("food positions must be consecutive")
		}
		amount, ok := positiveDecimal(input.Quantity.Amount)
		if !ok {
			return nil, invalid("invalid food quantity")
		}
		if input.FoodState != "RAW" && input.FoodState != "COOKED" && input.FoodState != "PREPARED" && input.FoodState != "UNKNOWN" {
			return nil, invalid("invalid food state")
		}
		var source foodSourceDoc
		var baseAmount, baseUnit string
		var provider map[string]foodNutrientDoc
		switch input.FoodType {
		case "USDA":
			if input.ProviderFoodId == "" || input.CustomFoodId != "" {
				return nil, invalid("invalid USDA food reference")
			}
			food, err := s.GetUSDAFood(ctx, &pb.GetUSDAFoodRequest{ProviderFoodId: input.ProviderFoodId})
			if err != nil {
				return nil, err
			}
			source = foodSourceDoc{Type: "USDA", ProviderFoodID: food.ProviderFoodId, Name: food.Name}
			baseAmount, baseUnit = food.BaseAmount, food.BaseUnit
			provider, err = nutritionDoc(food.Nutrients)
			if err != nil {
				return nil, err
			}
		case "CUSTOM":
			if input.CustomFoodId == "" || input.ProviderFoodId != "" {
				return nil, invalid("invalid custom food reference")
			}
			food, err := s.GetCustomFood(ctx, &pb.CustomFoodRequest{Id: input.CustomFoodId})
			if err != nil {
				return nil, err
			}
			source = foodSourceDoc{Type: "CUSTOM", CustomFoodID: food.Id, Name: food.Name}
			baseAmount, baseUnit = food.BaseQuantity.Amount, food.BaseQuantity.Unit
			provider, err = nutritionDoc(food.Nutrients)
			if err != nil {
				return nil, err
			}
		default:
			return nil, invalid("invalid food reference type")
		}
		if input.Quantity.Unit != baseUnit {
			return nil, invalid("serving unit must match the food base unit")
		}
		scaled, err := scaleNutrition(provider, input.Quantity.Amount, baseAmount)
		if err != nil {
			return nil, invalid("could not normalize food nutrients")
		}
		overrides, err := nutritionDoc(input.UserOverrideNutrition)
		if err != nil {
			return nil, err
		}
		final := make(map[string]foodNutrientDoc, len(scaled)+len(overrides))
		for key, value := range scaled {
			final[key] = value
		}
		for key, value := range overrides {
			if existing, exists := final[key]; exists && existing.Unit != value.Unit {
				return nil, invalid("override unit must match the provider nutrient")
			}
			final[key] = value
		}
		items = append(items, foodLogItemDoc{Position: index, FoodSource: source,
			Quantity: quantityDoc{Amount: amount, Unit: input.Quantity.Unit}, FoodState: input.FoodState,
			ProviderNutrition: scaled, UserOverrideNutrition: overrides, FinalNutrition: final})
	}
	aggregate, err := aggregateNutrition(items)
	if err != nil {
		return nil, status.Error(codes.Internal, "could not total nutrition")
	}
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, internal()
	}
	hash := sha256.Sum256(secret)
	now := time.Now().UTC()
	preview := foodPreviewDoc{ID: primitive.NewObjectID(), UserID: userID, SecretHash: hex.EncodeToString(hash[:]),
		Status: "OPEN", SourceType: "AD_HOC", LoggedAt: loggedAt, TrackingDay: day.Day, TimezoneAtLogging: day.Timezone,
		DailyResetTimeAtLog: day.DailyResetTime, TrackingDayRuleVersion: int(day.RuleVersion),
		MealCategory: req.MealCategory, Label: strings.TrimSpace(req.Label), Items: items,
		Aggregate: aggregate, CalculationVersion: foodCalculationVersion, ExpiresAt: now.Add(15 * time.Minute), CreatedAt: now}
	if _, err := s.DB.Collection("food_log_previews").InsertOne(ctx, preview); err != nil {
		return nil, internal()
	}
	return &pb.FoodGroupPreview{PreviewToken: preview.ID.Hex() + "." + base64.RawURLEncoding.EncodeToString(secret),
		ExpiresAt: preview.ExpiresAt.Format(time.RFC3339Nano), Group: previewGroupPB(preview)}, nil
}

func itemPB(item foodLogItemDoc) *pb.FoodLogItemSnapshot {
	id := ""
	if !item.ID.IsZero() {
		id = item.ID.Hex()
	}
	return &pb.FoodLogItemSnapshot{Id: id, Position: uint32(item.Position),
		FoodSource: &pb.FoodSource{Type: item.FoodSource.Type, ProviderFoodId: item.FoodSource.ProviderFoodID,
			CustomFoodId: item.FoodSource.CustomFoodID, Name: item.FoodSource.Name},
		Quantity:  &pb.FoodQuantity{Amount: item.Quantity.Amount.String(), Unit: item.Quantity.Unit},
		FoodState: item.FoodState, ProviderNutrition: nutrientPB(item.ProviderNutrition),
		UserOverrideNutrition: nutrientPB(item.UserOverrideNutrition), FinalNutrition: nutrientPB(item.FinalNutrition)}
}

func previewGroupPB(doc foodPreviewDoc) *pb.FoodLogGroup {
	items := make([]*pb.FoodLogItemSnapshot, 0, len(doc.Items))
	for _, item := range doc.Items {
		items = append(items, itemPB(item))
	}
	return &pb.FoodLogGroup{Status: "PREVIEW", LoggedAt: doc.LoggedAt.UTC().Format(time.RFC3339Nano),
		TrackingDay: doc.TrackingDay, TimezoneAtLogging: doc.TimezoneAtLogging,
		DailyResetTimeAtLogging: doc.DailyResetTimeAtLog, MealCategory: doc.MealCategory,
		Label: doc.Label, ItemCount: uint32(len(items)), Items: items,
		AggregateFinalNutrition: nutrientPB(doc.Aggregate), CalculationVersion: doc.CalculationVersion}
}
