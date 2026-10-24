package server

import "strings"

func summaryTargetLabel(value string) string {
	switch value {
	case "GENERAL_BENCHMARK":
		return "General reference"
	case "CUSTOM":
		return "Custom target"
	case "MACRO_DERIVED":
		return "Calculated from selected macros"
	case "PERSONALIZED", "CALCULATED":
		return "Calculated estimate"
	default:
		return "Selected target"
	}
}

// One selection controls presentation only. Source nutrition and snapshots stay complete.
var trackedNutrientCodes = strings.Fields("ENERGY PROTEIN CARBOHYDRATE FAT FIBER SUGARS SATURATED_FAT MONOUNSATURATED_FAT POLYUNSATURATED_FAT CHOLESTEROL CALCIUM IRON MAGNESIUM PHOSPHORUS POTASSIUM SODIUM ZINC SELENIUM VITAMIN_A VITAMIN_C VITAMIN_D VITAMIN_E VITAMIN_K THIAMIN RIBOFLAVIN NIACIN VITAMIN_B6 VITAMIN_B12 FOLATE_DFE SOLUBLE_FIBER INSOLUBLE_FIBER TRYPTOPHAN THREONINE ISOLEUCINE LEUCINE LYSINE METHIONINE CYSTINE PHENYLALANINE TYROSINE VALINE ARGININE HISTIDINE ALANINE ASPARTIC_ACID GLUTAMIC_ACID GLYCINE PROLINE SERINE")

func validateTrackedNutrients(values []string) error {
	allowed := map[string]bool{}
	for _, code := range trackedNutrientCodes {
		allowed[code] = true
	}
	seen := map[string]bool{}
	for _, code := range values {
		if !allowed[code] || seen[code] {
			return invalid("choose distinct supported nutrients")
		}
		seen[code] = true
	}
	if !seen["ENERGY"] {
		return invalid("energy must remain selected")
	}
	return nil
}
