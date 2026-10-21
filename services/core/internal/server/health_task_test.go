package server

import (
	"testing"
	"time"

	pb "github.com/architsangal/AteThat/services/core/internal/gen/atethat/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestHealthTaskValidation(t *testing.T) {
	for _, kind := range []string{"MEDICATION", "SUPPLEMENT", "TASK"} {
		req := &pb.UpsertHealthItemRequest{Type: kind, Name: "Brush teeth", StartDate: "2026-10-20"}
		if err := validateHealthItem(req); err != nil {
			t.Fatalf("%s without optional dosage rejected: %v", kind, err)
		}
	}
	for _, req := range []*pb.UpsertHealthItemRequest{
		{Type: "UNKNOWN", Name: "Brush teeth", StartDate: "2026-10-20"},
		{Type: "TASK", Name: "x", StartDate: "2026-10-20"},
		{Type: "TASK", Name: "Brush teeth", StartDate: "invalid"},
		{Type: "TASK", Name: "Brush teeth", StartDate: "2026-10-20", DosageAmountText: "1"},
		{Type: "TASK", Name: "Brush teeth", StartDate: "2026-10-20", DosageUnitText: "tablet"},
	} {
		if status.Code(validateHealthItem(req)) != codes.InvalidArgument {
			t.Fatalf("invalid task accepted: %+v", req)
		}
	}
}

func TestHealthTaskLifecycleIntegration(t *testing.T) {
	s, owner, other := phaseTwoServer(t)
	today := time.Now().UTC().Truncate(24 * time.Hour)
	start := today.AddDate(0, 0, -1).Format("2006-01-02")
	req := &pb.UpsertHealthItemRequest{Type: "TASK", Name: "Brush teeth", Instructions: "Morning and evening", Notes: "Synthetic task fixture", StartDate: start,
		Schedule: &pb.HealthSchedule{Timezone: "UTC", Frequency: "DAILY", Interval: 1, LocalTimes: []string{"08:00", "20:00"}, EffectiveFromDate: start}}
	item, err := s.CreateItem(owner, req)
	if err != nil || item.Type != "TASK" || item.DosageAmountText != "" || item.DosageUnitText != "" {
		t.Fatalf("create task: %v %v", item, err)
	}
	if _, err := s.GetItem(other, &pb.HealthItemRequest{Id: item.Id}); status.Code(err) != codes.NotFound {
		t.Fatalf("foreign task visible: %v", err)
	}
	query := &pb.ListOccurrencesRequest{From: today.AddDate(0, 0, -1).Format(time.RFC3339), To: today.Format(time.RFC3339), Limit: 100}
	occurrences, err := s.ListOccurrences(owner, query)
	if err != nil || len(occurrences.Items) != 2 || occurrences.Items[0].Type != "TASK" {
		t.Fatalf("task schedule: %v %v", occurrences, err)
	}
	first := occurrences.Items[0]
	if _, err := s.RecordOccurrence(other, &pb.RecordOccurrenceRequest{OccurrenceId: first.OccurrenceId, Status: "TAKEN", CompletedAt: first.ScheduledAt}); err == nil {
		t.Fatal("foreign task completion accepted")
	}
	done, err := s.RecordOccurrence(owner, &pb.RecordOccurrenceRequest{OccurrenceId: first.OccurrenceId, Status: "TAKEN", CompletedAt: first.ScheduledAt})
	if err != nil || done.Type != "TASK" || done.Status != "TAKEN" {
		t.Fatalf("task completion: %v %v", done, err)
	}
	second := occurrences.Items[1]
	if _, err := s.RecordOccurrence(owner, &pb.RecordOccurrenceRequest{OccurrenceId: second.OccurrenceId, Status: "SKIPPED"}); err != nil {
		t.Fatal(err)
	}
	req.Id, req.Version, req.Name, req.Schedule = item.Id, item.Version, "Apply solution", nil
	updated, err := s.UpdateItem(owner, req)
	if err != nil || updated.Name != "Apply solution" {
		t.Fatalf("rename task: %v %v", updated, err)
	}
	if _, err := s.UpdateItem(owner, req); status.Code(err) != codes.Aborted {
		t.Fatalf("stale task update accepted: %v", err)
	}
	history, err := s.ScheduleHistory(owner, &pb.ScheduleHistoryRequest{ItemId: item.Id, Limit: 10})
	if err != nil || len(history.Items) != 1 {
		t.Fatalf("task schedule history: %v %v", history, err)
	}
	if _, err := s.DeactivateItem(owner, &pb.DeactivateHealthItemRequest{Id: item.Id, Version: updated.Version, EndDate: today.Format("2006-01-02")}); err != nil {
		t.Fatal(err)
	}
	retained, err := s.ListOccurrences(owner, query)
	if err != nil || len(retained.Items) != 2 || retained.Items[0].Status != "TAKEN" || retained.Items[1].Status != "SKIPPED" {
		t.Fatalf("task history lost: %v %v", retained, err)
	}
}
