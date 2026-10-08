// Package trackingday is the authoritative conversion from a UTC instant to a
// user's local nutrition day. The resulting date is a business key, not an instant.
package trackingday

import (
	"errors"
	"fmt"
	"time"
)

const RuleVersion uint32 = 1

type Context struct {
	Day            string
	Timezone       string
	DailyResetTime string
	RuleVersion    uint32
	AsOfUTC        time.Time
}

// Derive compares local wall-clock time with the user's reset time. Comparing
// clock components (rather than adding 24 hours to an instant) keeps both sides
// of daylight-saving transitions on the correct local calendar day.
func Derive(instant time.Time, timezone, resetTime string) (Context, error) {
	if len(timezone) == 0 || len(timezone) > 100 || timezone == "Local" {
		return Context{}, errors.New("invalid timezone")
	}
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return Context{}, fmt.Errorf("invalid timezone: %w", err)
	}
	reset, err := time.Parse("15:04", resetTime)
	if err != nil || reset.Format("15:04") != resetTime {
		return Context{}, errors.New("invalid daily reset time")
	}
	utc := instant.UTC()
	local := utc.In(location)
	year, month, day := local.Date()
	if local.Hour()*60+local.Minute() < reset.Hour()*60+reset.Minute() {
		// Date arithmetic is performed on a date-only UTC value; it is not a
		// claim that local days are always 24 hours long.
		previous := time.Date(year, month, day, 12, 0, 0, 0, time.UTC).AddDate(0, 0, -1)
		year, month, day = previous.Date()
	}
	return Context{
		Day:            fmt.Sprintf("%04d-%02d-%02d", year, month, day),
		Timezone:       timezone,
		DailyResetTime: resetTime,
		RuleVersion:    RuleVersion,
		AsOfUTC:        utc,
	}, nil
}
