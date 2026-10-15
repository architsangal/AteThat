package server

import (
	pb "github.com/architsangal/AteThat/services/core/internal/gen/atethat/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"testing"
	"time"
)

func TestSchedulePauseResumeHistoryIntegration(t *testing.T) {
	s, owner, other := phaseTwoServer(t)
	today := time.Now().UTC().Truncate(24 * time.Hour)
	start := today.AddDate(0, 0, -3).Format("2006-01-02")
	request := &pb.UpsertHealthItemRequest{Type: "SUPPLEMENT", Name: "Local schedule test", StartDate: start, Schedule: &pb.HealthSchedule{Timezone: "UTC", Frequency: "DAILY", Interval: 1, LocalTimes: []string{"08:00"}, EffectiveFromDate: start, EffectiveToDate: today.AddDate(0, 0, -1).Format("2006-01-02")}}
	item, err := s.CreateItem(owner, request)
	if err != nil {
		t.Fatal(err)
	}
	query := &pb.ListOccurrencesRequest{From: today.AddDate(0, 0, -3).Format(time.RFC3339), To: today.Format(time.RFC3339), Limit: 100}
	before, err := s.ListOccurrences(owner, query)
	if err != nil || len(before.Items) != 3 {
		t.Fatal(before, err)
	}
	recorded, err := s.RecordOccurrence(owner, &pb.RecordOccurrenceRequest{OccurrenceId: before.Items[0].OccurrenceId, Status: "TAKEN", CompletedAt: before.Items[0].ScheduledAt})
	if err != nil {
		t.Fatal(err)
	}
	request.Id = item.Id
	request.Version = item.Version
	request.Schedule = &pb.HealthSchedule{Timezone: "UTC", Frequency: "PAUSED", Interval: 1, EffectiveFromDate: today.Format("2006-01-02")}
	if _, err := s.UpdateItem(owner, request); status.Code(err) != codes.InvalidArgument {
		t.Fatal("today's schedule rewrite accepted", err)
	}
	request.Schedule.EffectiveFromDate = today.AddDate(0, 0, 1).Format("2006-01-02")
	item, err = s.UpdateItem(owner, request)
	if err != nil {
		t.Fatal(err)
	}
	request.Version = item.Version
	request.Schedule = &pb.HealthSchedule{Timezone: "UTC", Frequency: "MONTHLY", Interval: 1, MonthDays: []int32{int32(today.AddDate(0, 0, 2).Day())}, LocalTimes: []string{"08:00"}, EffectiveFromDate: today.AddDate(0, 0, 2).Format("2006-01-02")}
	item, err = s.UpdateItem(owner, request)
	if err != nil || item.Schedule.Version != 3 {
		t.Fatal(item, err)
	}
	after, err := s.ListOccurrences(owner, query)
	if err != nil || len(after.Items) != len(before.Items) || after.Items[0].OccurrenceId != recorded.OccurrenceId || after.Items[0].Status != "TAKEN" {
		t.Fatal("past actions rewritten", after, err)
	}
	history, err := s.ScheduleHistory(owner, &pb.ScheduleHistoryRequest{ItemId: item.Id, Limit: 2})
	if err != nil || len(history.Items) != 2 || history.Items[0].Frequency != "MONTHLY" || history.Items[1].Frequency != "PAUSED" || history.NextBeforeVersion != 2 {
		t.Fatal(history, err)
	}
	history, err = s.ScheduleHistory(owner, &pb.ScheduleHistoryRequest{ItemId: item.Id, Limit: 2, BeforeVersion: history.NextBeforeVersion})
	if err != nil || len(history.Items) != 1 || history.Items[0].EffectiveToDate != today.AddDate(0, 0, -1).Format("2006-01-02") {
		t.Fatal("old ended schedule extended", history, err)
	}
	if _, err := s.ScheduleHistory(other, &pb.ScheduleHistoryRequest{ItemId: item.Id, Limit: 2}); status.Code(err) != codes.NotFound {
		t.Fatal("foreign history visible", err)
	}
	query.From = today.AddDate(0, 0, 1).Format(time.RFC3339)
	query.To = today.AddDate(0, 0, 2).Format(time.RFC3339)
	paused, err := s.ListOccurrences(owner, query)
	if err != nil || len(paused.Items) != 0 {
		t.Fatal("pause still creates occurrences", paused, err)
	}
	query.From = query.To
	query.To = today.AddDate(0, 0, 3).Format(time.RFC3339)
	resumed, err := s.ListOccurrences(owner, query)
	if err != nil || len(resumed.Items) != 1 || resumed.Items[0].ScheduleVersion != 3 {
		t.Fatal("resume missing", resumed, err)
	}
}
