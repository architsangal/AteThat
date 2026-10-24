package foods

import "testing"

func TestCanonicalProviderNumbers(t *testing.T) {
	for raw, want := range map[string]string{"0E-8": "0", "-0.000": "0", "286.00000000": "286", "2.143e1": "21.43", "1E-8": "0.00000001", "1.23000000000000000000000000001": "1.23000000000000000000000000001"} {
		got, ok := CanonicalNutrientAmount(raw)
		if !ok || got != want {
			t.Errorf("%q: %q %v, want %q", raw, got, ok, want)
		}
	}
	for _, raw := range []string{"-0.1", "NaN", "Infinity", "1/2", "1e99999999", "1e129", "1e-129", "1e32", "1e-32", "", "+2", ".5"} {
		if got, ok := CanonicalNutrientAmount(raw); ok {
			t.Errorf("accepted %q as %q", raw, got)
		}
	}
}

func TestScientificSearchAndAdditionalNutrients(t *testing.T) {
	data := []byte(`{"totalHits":1,"foods":[{"fdcId":123,"description":"Food","foodNutrients":[{"nutrientId":1008,"unitName":"KCAL","value":1.2e2},{"nutrientId":1079,"unitName":"G","value":0E-8},{"nutrientId":99999,"nutrientName":"Tryptophan","unitName":"G","value":2.5e-2}]}]}`)
	items, _, err := DecodeSearch(data)
	if err != nil {
		t.Fatal(err)
	}
	for code, want := range map[string]string{"ENERGY": "120", "FIBER": "0", "TRYPTOPHAN": "0.025"} {
		if items[0].Nutrients[code].Amount != want {
			t.Fatal(code, items[0].Nutrients[code])
		}
	}
	if items[0].Nutrients["VITAMIN_C"].Status != "UNKNOWN" {
		t.Fatal("missing nutrient became zero")
	}
}
