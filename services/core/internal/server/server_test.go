package server

import (
	"testing"

	pb "github.com/architsangal/AteThat/services/core/internal/gen/atethat/v1"
)

func TestValidation(t *testing.T) {
	if normalized, ok := validEmail("Person+tag@Example.com"); !ok || normalized != "person+tag@example.com" {
		t.Fatal("email normalization failed")
	}
	if _, ok := validEmail("bad\r\nBcc: attacker@example.com"); ok {
		t.Fatal("email header injection accepted")
	}
	if !validTimezone("Asia/Kolkata") || validTimezone("Mars/Unknown") {
		t.Fatal("timezone validation failed")
	}
	if validPassword("short") || !validPassword("longer safe password") {
		t.Fatal("password policy failed")
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
