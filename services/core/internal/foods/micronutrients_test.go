package foods

import "testing"

func TestNamedComponentsRequireMatchingUnitAndPreserveUnknown(t *testing.T) {
	item, err := DecodeDetail([]byte(`{"fdcId":123,"description":"Metadata fixture","foodNutrients":[{"nutrient":{"id":1210,"name":"Tryptophan","unitName":"G"},"amount":0.25},{"nutrient":{"id":1211,"name":"Threonine","unitName":"MG"},"amount":100},{"nutrient":{"id":1082,"name":"Fiber, soluble","unitName":"G"},"amount":0}]}`))
	if err != nil || item.Nutrients["TRYPTOPHAN"].Amount != "0.25" || item.Nutrients["SOLUBLE_FIBER"].Amount != "0" || item.Nutrients["THREONINE"].Status != "UNKNOWN" || item.Nutrients["LYSINE"].Status != "UNKNOWN" {
		t.Fatal(item, err)
	}
	items, _, err := DecodeSearch([]byte(`{"foods":[{"fdcId":1,"description":"Search fixture","foodNutrients":[{"nutrientId":1210,"nutrientName":"Tryptophan","unitName":"G","value":0.3}]}]}`))
	if err != nil || items[0].Nutrients["TRYPTOPHAN"].Amount != "0.3" {
		t.Fatal(items, err)
	}
}

func TestMicronutrientsAndUnits(t *testing.T) {
	item, err := DecodeDetail([]byte(`{"fdcId":123,"description":"Test food","foodNutrients":[{"nutrient":{"id":1087,"unitName":"mg"},"amount":0},{"nutrient":{"id":1114,"unitName":"UG"},"amount":2.5},{"nutrient":{"id":1106,"unitName":"IU"},"amount":100}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if item.Nutrients["CALCIUM"].Status != "KNOWN" || item.Nutrients["CALCIUM"].Amount != "0" {
		t.Fatal("known zero was lost")
	}
	if item.Nutrients["VITAMIN_D"].Amount != "2.5" || item.Nutrients["VITAMIN_D"].Unit != "mcg" {
		t.Fatal("microgram mapping failed")
	}
	if item.Nutrients["VITAMIN_A"].Status != "UNKNOWN" || item.Nutrients["IRON"].Status != "UNKNOWN" {
		t.Fatal("incompatible units or missing values became known")
	}
}
