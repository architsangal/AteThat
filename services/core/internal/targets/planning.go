package targets

import (
	"errors"
	"math/big"
)

const DefaultHeightCM = "175"
const DefaultWeightKG = "70.4375"
const DefaultActivity = "MODERATE"
const PlanningPolicyVersion = "planning-2026-10-21-v1"
const MismatchToleranceKcal = "1"

// MacroCalories is a planning convention only, never a recalculation of food energy.
func MacroCalories(values map[string]string) (string, error) {
	total := new(big.Rat)
	for key, factor := range map[string]int64{"PROTEIN": 4, "CARBOHYDRATE": 4, "FAT": 9} {
		text, exists := values[key]
		if !exists || !validAmount(text) {
			return "", errors.New("protein, carbohydrate and fat require nonnegative decimal values")
		}
		n, _ := new(big.Rat).SetString(text)
		total.Add(total, new(big.Rat).Mul(n, big.NewRat(factor, 1)))
	}
	return decimalText(total), nil
}

func decimalText(n *big.Rat) string {
	// Inputs have at most 32 characters; integer multiplication adds no new scale.
	s := n.FloatString(32)
	for len(s) > 0 && s[len(s)-1] == '0' {
		s = s[:len(s)-1]
	}
	if s[len(s)-1] == '.' {
		s = s[:len(s)-1]
	}
	return s
}

func ValidSelectedAmount(key, value string) bool {
	if !validAmount(value) {
		return false
	}
	n, _ := new(big.Rat).SetString(value)
	return key != "ENERGY" || n.Sign() > 0
}
