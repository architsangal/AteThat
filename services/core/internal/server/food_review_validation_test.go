package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	pb "github.com/architsangal/AteThat/services/core/internal/gen/atethat/v1"
)

func TestPaneerReview200Grams(t *testing.T) {
	// Public FDC 2053888 detail fields, including the original USDA 0E-8
	// numeric spellings. Parsing and re-serializing through floats hides the bug.
	s, owner, _ := phaseTwoServer(t)
	data, err := os.ReadFile("testdata/usda-paneer-2053888.json")
	if err != nil {
		t.Fatal(err)
	}
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}))
	defer provider.Close()
	s.Cfg.USDAAPIBaseURL = provider.URL
	req := &pb.PreviewFoodGroupRequest{LoggedAt: time.Now().UTC().Format(time.RFC3339), MealCategory: "DINNER", Items: []*pb.FoodLogItemInput{{FoodType: "USDA", ProviderFoodId: "2053888", FoodState: "UNKNOWN", Quantity: &pb.FoodQuantity{Amount: "200", Unit: "g"}}}}
	result, err := s.Preview(owner, req)
	if err != nil {
		t.Fatal(err)
	}
	if result.Group.AggregateFinalNutrition["ENERGY"].Amount != "572" {
		t.Fatal(result.Group.AggregateFinalNutrition)
	}
	for code, want := range map[string]string{"PROTEIN": "42.86", "FAT": "35.72", "CARBOHYDRATE": "7.14", "FIBER": "0", "IRON": "0", "VITAMIN_C": "0"} {
		value := result.Group.AggregateFinalNutrition[code]
		if value == nil || value.Status != "KNOWN" || value.Amount != want {
			t.Fatal(code, value)
		}
	}
	if result.Group.AggregateFinalNutrition["VITAMIN_B12"].Status != "UNKNOWN" {
		t.Fatal("missing nutrient became zero")
	}
	// A raw USDA response already cached before the fix is normalized on read;
	// repairing previews must not require purging the cache or another API call.
	provider.Close()
	if _, err := s.Preview(owner, req); err != nil {
		t.Fatal("cached source rejected", err)
	}
}
