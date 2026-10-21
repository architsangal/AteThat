package targets

import (
	"math/big"
	"testing"
)

func TestCalculatedEnergyBranches(t *testing.T) {
	// Independently transcribed Table 5-16 results for age 30, 180 cm, 100 kg.
	for _, tc := range []struct{ sex, activity, energy, rate string }{
		{"MALE", "SEDENTARY", "3008.17", "0.8"}, {"MALE", "LIGHT", "3244.57", "0.8"},
		{"MALE", "MODERATE", "3444.52", "1.2"}, {"MALE", "ACTIVE", "3444.52", "1.6"}, {"MALE", "VERY_ACTIVE", "3878.02", "2.0"},
		{"FEMALE", "SEDENTARY", "2575.2", "0.8"}, {"FEMALE", "LIGHT", "2767.47", "0.8"},
		{"FEMALE", "MODERATE", "2911.15", "1.2"}, {"FEMALE", "ACTIVE", "2911.15", "1.6"}, {"FEMALE", "VERY_ACTIVE", "3190.13", "2.0"},
	} {
		c, e := CalculateDefaults(30, tc.sex, "180", "100", tc.activity)
		if e != nil || c.Values["ENERGY"] != tc.energy || c.ProteinRate != tc.rate {
			t.Fatalf("%+v: %+v %v", tc, c, e)
		}
		macro, e := MacroCalories(c.Values)
		if e != nil {
			t.Fatal(e)
		}
		got, _ := new(big.Rat).SetString(macro)
		want, _ := new(big.Rat).SetString(tc.energy)
		difference := new(big.Rat).Abs(new(big.Rat).Sub(got, want))
		if difference.Cmp(big.NewRat(1, 100000000)) > 0 {
			t.Fatal("inconsistent energy", macro, tc.energy)
		}
	}
}
func TestCalculatedDefaultsEligibilityAndFallbacks(t *testing.T) {
	c, e := CalculateDefaults(26, "MALE", DefaultHeightCM, DefaultWeightKG, DefaultActivity)
	if e != nil || c.ProteinRate != "1.2" || c.Values["PROTEIN"] != "84.525" {
		t.Fatal(c, e)
	}
	for _, tc := range []struct {
		age          uint32
		sex, h, w, a string
	}{
		{18, "MALE", "175", "70", "MODERATE"}, {26, "DECLINED", "175", "70", "MODERATE"},
		{26, "", "175", "70", "MODERATE"}, {26, "MALE", "0", "70", "MODERATE"},
		{26, "MALE", "175", "-1", "MODERATE"}, {26, "MALE", "175", "70", "UNKNOWN"},
		{26, "MALE", "175", "10000", "VERY_ACTIVE"},
	} {
		if _, e := CalculateDefaults(tc.age, tc.sex, tc.h, tc.w, tc.a); e == nil {
			t.Fatal("accepted invalid inputs", tc)
		}
	}
}
