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
	NutrientID   int         `json:"nutrientId"`
	NutrientName string      `json:"nutrientName"`
	UnitName     string      `json:"unitName"`
	Value        json.Number `json:"value"`
	Nutrient     struct {
		ID       int    `json:"id"`
		Name     string `json:"name"`
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
	// Nutrient IDs and units follow USDA's FNDDS mapping:
	// https://www.ars.usda.gov/ARSUserFiles/80400530/pdf/fndds/2019_2020_FNDDS_Doc.pdf
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
		{"SUGARS", []int{2000}, "G"},
		{"SATURATED_FAT", []int{1258}, "G"},
		{"MONOUNSATURATED_FAT", []int{1292}, "G"},
		{"POLYUNSATURATED_FAT", []int{1293}, "G"},
		{"CHOLESTEROL", []int{1253}, "MG"},
		{"CALCIUM", []int{1087}, "MG"},
		{"IRON", []int{1089}, "MG"},
		{"MAGNESIUM", []int{1090}, "MG"},
		{"PHOSPHORUS", []int{1091}, "MG"},
		{"POTASSIUM", []int{1092}, "MG"},
		{"SODIUM", []int{1093}, "MG"},
		{"ZINC", []int{1095}, "MG"},
		{"SELENIUM", []int{1103}, "UG"},
		{"VITAMIN_A", []int{1106}, "UG"},
		{"VITAMIN_C", []int{1162}, "MG"},
		{"VITAMIN_D", []int{1114}, "UG"},
		{"VITAMIN_E", []int{1109}, "MG"},
		{"VITAMIN_K", []int{1185}, "UG"},
		{"THIAMIN", []int{1165}, "MG"},
		{"RIBOFLAVIN", []int{1166}, "MG"},
		{"NIACIN", []int{1167}, "MG"},
		{"VITAMIN_B6", []int{1175}, "MG"},
		{"VITAMIN_B12", []int{1178}, "UG"},
		{"FOLATE_DFE", []int{1190}, "UG"},
	} {
		unit := strings.ToLower(selection.unit)
		if unit == "ug" {
			unit = "mcg"
		}
		if _, exists := result.Nutrients[selection.key]; !exists {
			result.Nutrients[selection.key] = Nutrient{Status: "UNKNOWN", Unit: unit}
		}
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
	// Match these additional components by the explicit name in USDA metadata,
	// not guessed IDs. Only gram values are accepted; absence stays unknown.
	for name, key := range map[string]string{
		"Tryptophan": "TRYPTOPHAN", "Threonine": "THREONINE", "Isoleucine": "ISOLEUCINE", "Leucine": "LEUCINE",
		"Lysine": "LYSINE", "Methionine": "METHIONINE", "Cystine": "CYSTINE", "Phenylalanine": "PHENYLALANINE",
		"Tyrosine": "TYROSINE", "Valine": "VALINE", "Arginine": "ARGININE", "Histidine": "HISTIDINE",
		"Alanine": "ALANINE", "Aspartic acid": "ASPARTIC_ACID", "Glutamic acid": "GLUTAMIC_ACID",
		"Glycine": "GLYCINE", "Proline": "PROLINE", "Serine": "SERINE",
		"Fiber, soluble": "SOLUBLE_FIBER", "Fiber, insoluble": "INSOLUBLE_FIBER",
	} {
		result.Nutrients[key] = Nutrient{Status: "UNKNOWN", Unit: "g"}
		for _, item := range raw.FoodNutrients {
			valueName, valueUnit, value := item.NutrientName, item.UnitName, item.Value
			if item.NutrientID == 0 {
				valueName, valueUnit, value = item.Nutrient.Name, item.Nutrient.UnitName, item.Amount
			}
			if !strings.EqualFold(strings.TrimSpace(valueName), name) || !strings.EqualFold(valueUnit, "G") || value == "" {
				continue
			}
			if number, err := strconv.ParseFloat(value.String(), 64); err == nil && number >= 0 {
				result.Nutrients[key] = Nutrient{Status: "KNOWN", Amount: value.String(), Unit: "g"}
				break
			}
		}
	}
	return result
}
