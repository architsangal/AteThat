package server

import (
	"testing"
	"time"

	pb "github.com/architsangal/AteThat/services/core/internal/gen/atethat/v1"
)

func TestValidation(t *testing.T) {
	if normalized, ok := validEmail("Person+tag@Example.com"); !ok || normalized != "person+tag@example.com" {
		t.Fatal("email normalization failed")
	}
	if _, ok := validEmail("bad\r\nBcc: attacker@example.com"); ok {
		t.Fatal("email header injection accepted")
	}
	if !validTimezone("Asia/Kolkata") || validTimezone("Mars/Unknown") || validTimezone("Local") {
		t.Fatal("timezone validation failed")
	}
	if validPassword("short") || !validPassword("longer safe password") {
		t.Fatal("password policy failed")
	}
}

func TestAnalysisRangeAndTimezoneBounds(t *testing.T) {
	if _, _, err := parseDayRange("2026-10-01", "2026-10-08"); err != nil {
		t.Fatalf("valid range rejected: %v", err)
	}
	for _, pair := range [][2]string{{"2026-10-08", "2026-10-01"}, {"2026-02-30", "2026-03-01"}, {"2025-01-01", "2026-10-08"}} {
		if _, _, err := parseDayRange(pair[0], pair[1]); err == nil {
			t.Errorf("invalid range accepted: %v", pair)
		}
	}
	start, end, err := summaryBounds(dailyNutritionSummaryDoc{TrackingDay: "2026-10-08", TimezoneAtSummary: "Asia/Kolkata", DailyResetTime: "04:00"})
	if err != nil || !start.Equal(time.Date(2026, 10, 7, 22, 30, 0, 0, time.UTC)) || end.Sub(start) != 24*time.Hour {
		t.Fatalf("incorrect UTC Tracking Day bounds: %v %v %v", start, end, err)
	}
}

func TestQuantityUsesDecimal128(t *testing.T) {
	quantity, err := parseQuantity(&pb.DecimalQuantity{Amount: "72.50", Unit: "kg"})
	if err != nil || quantity.Amount.String() != "72.50" {
		t.Fatalf("decimal conversion failed: %v", err)
	}
	if _, err := parseQuantity(&pb.DecimalQuantity{Amount: "NaN", Unit: "kg"}); err == nil {
		t.Fatal("NaN accepted")
	}
	if _, err := parseQuantity(&pb.DecimalQuantity{Amount: "-1", Unit: "kg"}); err == nil {
		t.Fatal("negative amount accepted")
	}
}
