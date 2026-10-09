package health

import (
	"testing"
	"time"
)

func TestDailyAcrossTimezone(t *testing.T) {
	s := Schedule{Timezone: "Asia/Kolkata", Frequency: "DAILY", Interval: 1, LocalTimes: []string{"08:00"}, EffectiveFromDate: "2026-10-08"}
	from, _ := time.Parse(time.RFC3339, "2026-10-08T00:00:00Z")
	to, _ := time.Parse(time.RFC3339, "2026-10-09T00:00:00Z")
	items, err := Expand(s, from, to)
	if err != nil || len(items) != 1 || items[0].At.Format(time.RFC3339) != "2026-10-08T02:30:00Z" {
		t.Fatalf("occurrences: %+v %v", items, err)
	}
}

func TestAlternateDay(t *testing.T) {
	s := Schedule{Timezone: "UTC", Frequency: "DAILY", Interval: 2, LocalTimes: []string{"08:00"}, EffectiveFromDate: "2026-10-08"}
	from, _ := time.Parse(time.RFC3339, "2026-10-08T00:00:00Z")
	to, _ := time.Parse(time.RFC3339, "2026-10-11T00:00:00Z")
	items, err := Expand(s, from, to)
	if err != nil || len(items) != 2 {
		t.Fatalf("occurrences: %+v %v", items, err)
	}
}

func TestEffectiveEndDateBoundsOccurrences(t *testing.T) {
	s := Schedule{Timezone: "UTC", Frequency: "DAILY", Interval: 1, LocalTimes: []string{"08:00"}, EffectiveFromDate: "2026-10-08", EffectiveToDate: "2026-10-09"}
	from, _ := time.Parse(time.RFC3339, "2026-10-08T00:00:00Z")
	to, _ := time.Parse(time.RFC3339, "2026-10-12T00:00:00Z")
	items, err := Expand(s, from, to)
	if err != nil || len(items) != 2 || items[1].LocalDate != "2026-10-09" {
		t.Fatalf("effective end date was not enforced: %+v %v", items, err)
	}
}
