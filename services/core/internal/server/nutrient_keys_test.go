package server

import (
	"strings"
	"testing"

	pb "github.com/architsangal/AteThat/services/core/internal/gen/atethat/v1"
)

func TestVitaminIdentifiersInCustomFoodAndSnapshot(t *testing.T) {
	for _, nutrientStatus := range []string{"KNOWN", "UNKNOWN"} {
		t.Run(nutrientStatus, func(t *testing.T) {
			amount := ""
			if nutrientStatus == "KNOWN" {
				amount = "0.25"
			}
			nutrients := map[string]*pb.FoodNutrient{
				"ENERGY":      {Status: "KNOWN", Amount: "200", Unit: "kcal"},
				"VITAMIN_B6":  {Status: nutrientStatus, Amount: amount, Unit: "mg"},
				"VITAMIN_B12": {Status: nutrientStatus, Amount: amount, Unit: "mcg"},
			}
			food, err := customFoodInput(&pb.UpsertCustomFoodRequest{
				Name: "Synthetic food", FoodState: "PREPARED",
				BaseQuantity: &pb.FoodQuantity{Amount: "100", Unit: "g"}, Nutrients: nutrients,
			})
			if err != nil {
				t.Fatalf("custom food rejected digit-bearing vitamin identifiers: %v", err)
			}
			snapshot, err := nutritionDoc(nutrients)
			if err != nil {
				t.Fatalf("nutrition snapshot rejected vitamin identifiers: %v", err)
			}
			for _, values := range []map[string]foodNutrientDoc{food.Nutrients, snapshot} {
				for _, key := range []string{"VITAMIN_B6", "VITAMIN_B12"} {
					value := values[key]
					if value.Status != nutrientStatus || (nutrientStatus == "UNKNOWN" && value.Amount != nil) ||
						(nutrientStatus == "KNOWN" && (value.Amount == nil || value.Amount.String() != amount)) {
						t.Fatalf("lost known/unknown vitamin value: %s %+v", key, value)
					}
				}
			}
		})
	}
}

func TestNutrientIdentifierRejectsUnsafeKeys(t *testing.T) {
	for _, key := range []string{"", "6VITAMIN", "$set", "VITAMIN.B6", "vitamin_b6", "VITAMIN-B6", "VITAMIN B6", strings.Repeat("A", 41)} {
		if nutrientKeyPattern.MatchString(key) {
			t.Errorf("accepted invalid nutrient identifier %q", key)
		}
	}
}
