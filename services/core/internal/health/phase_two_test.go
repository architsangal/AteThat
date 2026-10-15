package health

import (
	"testing"
	"time"
)

func TestMonthlyPauseAndBounds(t *testing.T) {
	s := Schedule{Timezone: "UTC", Frequency: "MONTHLY", Interval: 1, MonthDays: []int32{29, 31}, LocalTimes: []string{"08:00", "20:00"}, EffectiveFromDate: "2024-01-01"}
	from := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 1, 0)
	rows, err := Expand(s, from, to)
	if err != nil || len(rows) != 2 || rows[0].LocalDate != "2024-02-29" {
		t.Fatal("leap day / absent date", rows, err)
	}
	from = from.AddDate(1, 0, 0)
	to = from.AddDate(0, 1, 0)
	rows, err = Expand(s, from, to)
	if err != nil || len(rows) != 0 {
		t.Fatal("nonexistent date not skipped", rows, err)
	}
	s.Interval = 2
	from = time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	to = from.AddDate(0, 1, 0)
	rows, err = Expand(s, from, to)
	if err != nil || len(rows) != 0 {
		t.Fatal("monthly interval ignored", rows, err)
	}
	s.Interval = 1
	s.EffectiveToDate = "2024-02-28"
	rows, err = Expand(s, from, to)
	if err != nil || len(rows) != 0 {
		t.Fatal("end bound ignored", rows, err)
	}
	s.Frequency = "PAUSED"
	s.MonthDays = nil
	s.LocalTimes = nil
	rows, err = Expand(s, from, to)
	if err != nil || len(rows) != 0 {
		t.Fatal("pause creates occurrences", rows, err)
	}
	for _, days := range [][]int32{{0}, {32}, {1, 1}, nil} {
		s.Frequency = "MONTHLY"
		s.MonthDays = days
		s.LocalTimes = []string{"08:00"}
		if Validate(s) == nil {
			t.Fatal("invalid month dates accepted", days)
		}
	}
}
