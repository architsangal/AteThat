package foods

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// USDA reports food nutrients per 100 g for the supported Foundation, SR
// Legacy, Survey and Branded result types. Absence is unknown, never zero.
type Nutrient struct {
	Status string `json:"status"`
	Amount string `json:"amount,omitempty"`
	Unit   string `json:"unit"`
}

type Food struct {
	ID        string              `json:"provider_food_id"`
	Name      string              `json:"name"`
	DataType  string              `json:"data_type"`
	Nutrients map[string]Nutrient `json:"nutrients"`
}

type rawNutrient struct {
	NutrientID int         `json:"nutrientId"`
	UnitName   string      `json:"unitName"`
	Value      json.Number `json:"value"`
	Nutrient   struct {
		ID       int    `json:"id"`
		UnitName string `json:"unitName"`
	} `json:"nutrient"`
	Amount json.Number `json:"amount"`
}

type rawFood struct {
	FDCID         int64         `json:"fdcId"`
	Description   string        `json:"description"`
	DataType      string        `json:"dataType"`
	FoodNutrients []rawNutrient `json:"foodNutrients"`
}

type SearchResponse struct {
	TotalHits int64     `json:"totalHits"`
	Foods     []rawFood `json:"foods"`
}

func DecodeSearch(data []byte) ([]Food, int64, error) {
	var response SearchResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, 0, err
	}
	foods := make([]Food, 0, len(response.Foods))
	for _, item := range response.Foods {
		if item.FDCID > 0 && strings.TrimSpace(item.Description) != "" {
			foods = append(foods, Normalize(item))
		}
	}
	return foods, response.TotalHits, nil
}

func DecodeDetail(data []byte) (Food, error) {
	var raw rawFood
	if err := json.Unmarshal(data, &raw); err != nil {
		return Food{}, err
	}
	if raw.FDCID <= 0 || strings.TrimSpace(raw.Description) == "" {
		return Food{}, fmt.Errorf("USDA food is missing identity")
	}
	return Normalize(raw), nil
}

func Normalize(raw rawFood) Food {
	result := Food{ID: strconv.FormatInt(raw.FDCID, 10), Name: raw.Description,
		DataType: raw.DataType, Nutrients: map[string]Nutrient{}}
	for key, unit := range map[string]string{"ENERGY": "kcal", "PROTEIN": "g", "CARBOHYDRATE": "g", "FAT": "g", "FIBER": "g"} {
		result.Nutrients[key] = Nutrient{Status: "UNKNOWN", Unit: unit}
	}
	// Energy variants are alternatives, not additive. Prefer the current
	// metabolizable-energy record when USDA provides more than one.
	for _, selection := range []struct {
		key  string
		ids  []int
		unit string
	}{
		{"ENERGY", []int{2047, 2048, 1008}, "KCAL"},
		{"PROTEIN", []int{1003}, "G"},
		{"CARBOHYDRATE", []int{1005}, "G"},
		{"FAT", []int{1004}, "G"},
		{"FIBER", []int{1079}, "G"},
	} {
		for _, id := range selection.ids {
			found := false
			for _, item := range raw.FoodNutrients {
				valueID, valueUnit, value := item.NutrientID, item.UnitName, item.Value
				if valueID == 0 {
					valueID, valueUnit, value = item.Nutrient.ID, item.Nutrient.UnitName, item.Amount
				}
				if valueID != id || strings.ToUpper(valueUnit) != selection.unit || value == "" {
					continue
				}
				if number, err := strconv.ParseFloat(value.String(), 64); err == nil && number >= 0 {
					result.Nutrients[selection.key] = Nutrient{Status: "KNOWN", Amount: value.String(), Unit: result.Nutrients[selection.key].Unit}
					found = true
					break
				}
			}
			if found {
				break
			}
		}
	}
	return result
}
