package foods

import "testing"

func TestNormalizeKnownAndUnknown(t *testing.T) {
	data := []byte(`{"totalHits":1,"foods":[{"fdcId":123,"description":"Rice","dataType":"Foundation","foodNutrients":[{"nutrientId":1008,"unitName":"KCAL","value":130},{"nutrientId":1003,"unitName":"G","value":2.7}]}]}`)
	items, count, err := DecodeSearch(data)
	if err != nil || count != 1 || len(items) != 1 {
		t.Fatalf("decode: count=%d items=%d err=%v", count, len(items), err)
	}
	if items[0].Nutrients["ENERGY"].Amount != "130" || items[0].Nutrients["PROTEIN"].Amount != "2.7" || items[0].Nutrients["FIBER"].Status != "UNKNOWN" {
		t.Fatalf("incorrect nutrient mapping: %+v", items[0].Nutrients)
	}
}

func TestDetailNestedNutrientAndEnergyPriority(t *testing.T) {
	data := []byte(`{"fdcId":456,"description":"Food","foodNutrients":[{"nutrient":{"id":1008,"unitName":"KCAL"},"amount":190},{"nutrient":{"id":2047,"unitName":"KCAL"},"amount":200}]}`)
	item, err := DecodeDetail(data)
	if err != nil || item.Nutrients["ENERGY"].Amount != "200" {
		t.Fatalf("detail: %+v, %v", item, err)
	}
}
