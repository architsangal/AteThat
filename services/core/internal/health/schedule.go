package health

import (
	"errors"
	"sort"
	"time"
)

type Schedule struct {
	Timezone          string
	Frequency         string
	Interval          int
	Weekdays          []string
	LocalTimes        []string
	MonthDays         []int32
	EffectiveFromDate string
	EffectiveToDate   string
}

type Occurrence struct {
	At        time.Time
	LocalDate string
	LocalTime string
	Timezone  string
}

var weekdays = map[string]time.Weekday{"SUN": time.Sunday, "MON": time.Monday, "TUE": time.Tuesday,
	"WED": time.Wednesday, "THU": time.Thursday, "FRI": time.Friday, "SAT": time.Saturday}

func Validate(s Schedule) error {
	if _, err := time.LoadLocation(s.Timezone); err != nil {
		return errors.New("invalid schedule timezone")
	}
	if s.Interval < 1 || s.Interval > 365 {
		return errors.New("invalid recurrence interval")
	}
	if s.Frequency != "DAILY" && s.Frequency != "WEEKLY" && s.Frequency != "MONTHLY" && s.Frequency != "PAUSED" {
		return errors.New("invalid frequency")
	}
	if s.Frequency == "MONTHLY" {
		if len(s.MonthDays) == 0 || len(s.MonthDays) > 31 {
			return errors.New("monthly schedule requires calendar days")
		}
		seen := map[int32]bool{}
		for _, day := range s.MonthDays {
			if day < 1 || day > 31 || seen[day] {
				return errors.New("invalid or duplicate month day")
			}
			seen[day] = true
		}
	} else if len(s.MonthDays) > 0 {
		return errors.New("month days require monthly recurrence")
	}
	if s.Frequency != "WEEKLY" && len(s.Weekdays) > 0 {
		return errors.New("weekdays require weekly recurrence")
	}
	if s.Frequency == "DAILY" && len(s.Weekdays) != 0 {
		return errors.New("daily schedule must not include weekdays")
	}
	if s.Frequency == "WEEKLY" && (len(s.Weekdays) == 0 || len(s.Weekdays) > 7) {
		return errors.New("weekly schedule requires weekdays")
	}
	seen := map[string]bool{}
	for _, weekday := range s.Weekdays {
		if _, ok := weekdays[weekday]; !ok || seen[weekday] {
			return errors.New("invalid or duplicate weekday")
		}
		seen[weekday] = true
	}
	if (s.Frequency != "PAUSED" && len(s.LocalTimes) == 0) || len(s.LocalTimes) > 12 {
		return errors.New("schedule requires 1–12 local times")
	}
	seen = map[string]bool{}
	for _, localTime := range s.LocalTimes {
		parsed, err := time.Parse("15:04", localTime)
		if err != nil || parsed.Format("15:04") != localTime || seen[localTime] {
			return errors.New("invalid or duplicate local time")
		}
		seen[localTime] = true
	}
	start, err := time.Parse("2006-01-02", s.EffectiveFromDate)
	if err != nil || start.Format("2006-01-02") != s.EffectiveFromDate {
		return errors.New("invalid effective date")
	}
	if s.EffectiveToDate != "" {
		end, err := time.Parse("2006-01-02", s.EffectiveToDate)
		if err != nil || end.Before(start) {
			return errors.New("invalid effective end date")
		}
	}
	return nil
}

func Expand(s Schedule, from, to time.Time) ([]Occurrence, error) {
	if err := Validate(s); err != nil {
		return nil, err
	}
	if !from.Before(to) || to.Sub(from) > 32*24*time.Hour {
		return nil, errors.New("occurrence range must be at most 32 days")
	}
	if s.Frequency == "PAUSED" {
		return []Occurrence{}, nil
	}
	location, _ := time.LoadLocation(s.Timezone)
	anchor, _ := time.Parse("2006-01-02", s.EffectiveFromDate)
	firstLocal := from.In(location)
	lastLocal := to.In(location)
	first := time.Date(firstLocal.Year(), firstLocal.Month(), firstLocal.Day(), 12, 0, 0, 0, time.UTC).AddDate(0, 0, -1)
	last := time.Date(lastLocal.Year(), lastLocal.Month(), lastLocal.Day(), 12, 0, 0, 0, time.UTC).AddDate(0, 0, 1)
	result := []Occurrence{}
	for date := first; !date.After(last); date = date.AddDate(0, 0, 1) {
		dateText := date.Format("2006-01-02")
		if dateText < s.EffectiveFromDate || (s.EffectiveToDate != "" && dateText > s.EffectiveToDate) {
			continue
		}
		days := int(date.Sub(anchor).Hours() / 24)
		if days < 0 {
			continue
		}
		eligible := false
		if s.Frequency == "DAILY" {
			eligible = days%s.Interval == 0
		} else if s.Frequency == "MONTHLY" {
			months := (date.Year()-anchor.Year())*12 + int(date.Month()-anchor.Month())
			if months%s.Interval == 0 {
				for _, day := range s.MonthDays {
					if date.Day() == int(day) {
						eligible = true
					}
				}
			}
		} else {
			anchorWeek := anchor.AddDate(0, 0, -int((int(anchor.Weekday())+6)%7))
			weeks := int(date.Sub(anchorWeek).Hours()/24) / 7
			if weeks%s.Interval == 0 {
				for _, weekday := range s.Weekdays {
					if date.Weekday() == weekdays[weekday] {
						eligible = true
						break
					}
				}
			}
		}
		if !eligible {
			continue
		}
		for _, clock := range s.LocalTimes {
			parsed, _ := time.Parse("15:04", clock)
			at := time.Date(date.Year(), date.Month(), date.Day(), parsed.Hour(), parsed.Minute(), 0, 0, location)
			wall := at.In(location)
			// An omitted spring-forward time cannot be silently shifted.
			if wall.Format("2006-01-02") != dateText || wall.Format("15:04") != clock {
				continue
			}
			if at.Before(from) || !at.Before(to) {
				continue
			}
			result = append(result, Occurrence{At: at.UTC(), LocalDate: dateText, LocalTime: clock, Timezone: s.Timezone})
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].At.Before(result[j].At) })
	return result, nil
}
