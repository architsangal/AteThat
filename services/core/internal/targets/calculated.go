package targets

import (
	"errors"
	"fmt"
	"math/big"
	"strings"
)

const CalculatedVersion = "adult-eer-2023-activity-v1"
const EnergySourceURL = "https://www.nationalacademies.org/read/26818/chapter/7"

type Calculation struct {
	Values                                  map[string]string
	Equation, ProteinRate, ActivityCategory string
}

// EER coefficients: National Academies (2023), Table 5-16. Activity is
// already incorporated in the selected equation; never multiply it again.
var adultEquations = map[string]map[string][4]string{
	"MALE": {
		"Inactive":    {"753.07", "-10.83", "6.50", "14.10"},
		"Low active":  {"581.47", "-10.83", "8.30", "14.94"},
		"Active":      {"1004.82", "-10.83", "6.52", "15.91"},
		"Very active": {"-517.88", "-10.83", "15.61", "19.11"},
	},
	"FEMALE": {
		"Inactive":    {"584.90", "-7.01", "5.72", "11.71"},
		"Low active":  {"575.77", "-7.01", "6.60", "12.14"},
		"Active":      {"710.25", "-7.01", "6.54", "12.34"},
		"Very active": {"511.83", "-7.01", "9.07", "12.56"},
	},
}

// Product approximations, not a source-defined conversion of an activity
// label into a measured PAL or a clinical protein prescription.
var activityDefaults = map[string]struct{ category, protein string }{
	"SEDENTARY": {"Inactive", "0.8"}, "LIGHT": {"Low active", "0.8"},
	"MODERATE": {"Active", "1.2"}, "ACTIVE": {"Active", "1.6"},
	"VERY_ACTIVE": {"Very active", "2.0"},
}

func CalculateDefaults(age uint32, sex, height, weight, activity string) (Calculation, error) {
	result := Calculation{}
	if age < 19 {
		return result, errors.New("Calculated defaults are available for adults aged 19 or older. You can still choose targets manually.")
	}
	equations, ok := adultEquations[sex]
	if !ok {
		return result, errors.New("Add date of birth and select male or female for the energy equation in your profile, or keep using manual targets. Sex is never inferred.")
	}
	choice, ok := activityDefaults[activity]
	if !ok {
		return result, errors.New("Choose a supported activity level in your profile.")
	}
	if !ValidSelectedAmount("ENERGY", height) || !ValidSelectedAmount("ENERGY", weight) {
		return result, errors.New("Height and profile weight must be positive.")
	}
	rat := func(s string) *big.Rat { n, _ := new(big.Rat).SetString(s); return n }
	h, w := rat(height), rat(weight)
	c := equations[choice.category]
	energy := rat(c[0])
	for i, input := range []*big.Rat{new(big.Rat).SetInt64(int64(age)), h, w} {
		energy.Add(energy, new(big.Rat).Mul(rat(c[i+1]), input))
	}
	if energy.Sign() <= 0 {
		return result, errors.New("These profile details do not produce a positive energy estimate. Review your profile or choose targets manually.")
	}
	protein := new(big.Rat).Mul(w, rat(choice.protein))
	fatEnergy := new(big.Rat).Mul(energy, rat("0.275"))
	carbEnergy := new(big.Rat).Sub(energy, new(big.Rat).Add(new(big.Rat).Mul(protein, rat("4")), fatEnergy))
	carbShare := new(big.Rat).Quo(carbEnergy, energy)
	if carbShare.Cmp(rat("0.45")) < 0 || carbShare.Cmp(rat("0.65")) > 0 {
		return result, errors.New("These inputs produce a carbohydrate share outside the adult reference range (45–65%). Review profile details or choose targets manually; no values have been changed.")
	}
	// Sixteen fractional digits keep calculated values within Decimal128.
	canonical := func(n *big.Rat) string { return strings.TrimRight(strings.TrimRight(n.FloatString(16), "0"), ".") }
	result.Values = map[string]string{"ENERGY": canonical(energy), "PROTEIN": canonical(protein), "FAT": canonical(new(big.Rat).Quo(fatEnergy, rat("9"))), "CARBOHYDRATE": canonical(new(big.Rat).Quo(carbEnergy, rat("4"))), "FIBER": canonical(new(big.Rat).Mul(energy, rat("0.014")))}
	result.Equation = fmt.Sprintf("%s + (%s × age) + (%s × height in cm) + (%s × profile weight in kg)", c[0], c[1], c[2], c[3])
	result.ProteinRate, result.ActivityCategory = choice.protein, choice.category
	return result, nil
}
