package trackingday

import (
	"testing"
	"time"
)

func TestDerive(t *testing.T) {
	cases := []struct {
		name, instant, timezone, reset, want string
	}{
		{"before reset IST", "2026-10-07T22:29:59Z", "Asia/Kolkata", "04:00", "2026-10-07"},
		{"at reset IST", "2026-10-07T22:30:00Z", "Asia/Kolkata", "04:00", "2026-10-08"},
		{"year boundary", "2026-01-01T01:00:00Z", "UTC", "04:00", "2025-12-31"},
		{"leap day", "2024-03-01T01:00:00Z", "UTC", "04:00", "2024-02-29"},
		{"spring DST before", "2026-03-08T06:30:00Z", "America/New_York", "04:00", "2026-03-07"},
		{"spring DST after", "2026-03-08T08:00:00Z", "America/New_York", "04:00", "2026-03-08"},
		{"fall DST first repeated hour", "2026-11-01T05:30:00Z", "America/New_York", "01:45", "2026-10-31"},
		{"fall DST second repeated hour", "2026-11-01T06:30:00Z", "America/New_York", "01:45", "2026-10-31"},
		{"reset changed", "2026-10-08T03:00:00Z", "UTC", "02:00", "2026-10-08"},
		{"reset changed later", "2026-10-08T03:00:00Z", "UTC", "04:00", "2026-10-07"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			instant, err := time.Parse(time.RFC3339, tc.instant)
			if err != nil {
				t.Fatal(err)
			}
			got, err := Derive(instant, tc.timezone, tc.reset)
			if err != nil || got.Day != tc.want || got.RuleVersion != RuleVersion || got.AsOfUTC.Location() != time.UTC {
				t.Fatalf("Derive() = %+v, %v; want day %s", got, err, tc.want)
			}
		})
	}
}

func TestDeriveRejectsInvalidConfiguration(t *testing.T) {
	for _, tc := range []struct{ zone, reset string }{
		{"Mars/Unknown", "04:00"}, {"Local", "04:00"}, {"UTC", "24:00"}, {"UTC", "4:00"},
	} {
		if _, err := Derive(time.Now(), tc.zone, tc.reset); err == nil {
			t.Fatalf("accepted zone %q reset %q", tc.zone, tc.reset)
		}
	}
}
