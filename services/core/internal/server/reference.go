package server

import (
	"context"
	"time"

	"github.com/architsangal/AteThat/services/core/internal/targets"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func ensureFDADailyValues(ctx context.Context, s *Server) error {
	set, values := targets.FDADailyValues()
	stored := make(map[string]referenceValueDoc, len(values))
	for key, value := range values {
		amount, err := primitive.ParseDecimal128(value.Amount)
		if err != nil {
			return err
		}
		stored[key] = referenceValueDoc{Amount: amount, Unit: value.Unit}
	}
	doc := referenceSetDoc{ID: primitive.NewObjectID(), ReferenceKey: set.Key,
		ReferenceVersion: set.Version, Status: "ACTIVE", TargetBasis: set.Basis,
		Source: set.Source, RequiredKeys: set.RequiredNutrients, Values: stored, CreatedAt: time.Now().UTC()}
	_, err := s.DB.Collection("nutrition_reference_sets").UpdateOne(ctx,
		bson.M{"reference_key": set.Key, "reference_version": set.Version},
		bson.M{"$setOnInsert": doc}, options.Update().SetUpsert(true))
	return err
}
