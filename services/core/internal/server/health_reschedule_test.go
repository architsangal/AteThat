package server

import (
	pb "github.com/architsangal/AteThat/services/core/internal/gen/atethat/v1"
	"go.mongodb.org/mongo-driver/bson"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"testing"
	"time"
)

func TestRescheduleMovesEffectiveRangePreservesIdentityAndAudit(t *testing.T) {
	s, owner, other := phaseTwoServer(t)
	if err := createIndexes(owner, s.DB); err != nil {
		t.Fatal(err)
	}
	today := time.Now().UTC().Truncate(24 * time.Hour)
	oldDay := today.AddDate(0, 0, -1)
	item, err := s.CreateItem(owner, &pb.UpsertHealthItemRequest{Type: "TASK", Name: "Reschedule fixture", StartDate: oldDay.Format("2006-01-02"), Schedule: &pb.HealthSchedule{Timezone: "UTC", Frequency: "DAILY", Interval: 1, LocalTimes: []string{"08:00"}, EffectiveFromDate: oldDay.Format("2006-01-02")}})
	if err != nil || item.Type != "TASK" {
		t.Fatal(item, err)
	}
	query := func(from, to time.Time, limit uint32) *pb.ListOccurrencesRequest {
		return &pb.ListOccurrencesRequest{From: from.Format(time.RFC3339), To: to.Format(time.RFC3339), Limit: limit}
	}
	old, err := s.ListOccurrences(owner, query(oldDay, today, 100))
	if err != nil || len(old.Items) != 1 {
		t.Fatal(old, err)
	}
	id := old.Items[0].OccurrenceId
	zero := uint64(0)
	skipped, err := s.RecordOccurrence(owner, &pb.RecordOccurrenceRequest{OccurrenceId: id, Status: "SKIPPED", ExpectedEventVersion: &zero})
	if err != nil {
		t.Fatal(err)
	}
	destination := today.AddDate(0, 0, 2).Add(9 * time.Hour)
	req := &pb.RecordOccurrenceRequest{OccurrenceId: id, Status: "RESCHEDULED", RescheduledTo: destination.Format(time.RFC3339), ExpectedEventVersion: &skipped.EventVersion}
	if _, err = s.RecordOccurrence(other, req); status.Code(err) != codes.NotFound {
		t.Fatal("foreign move", err)
	}
	moved, err := s.RecordOccurrence(owner, req)
	if err != nil || moved.Status != "SCHEDULED" || moved.OccurrenceId != id || moved.OriginalScheduledAt != old.Items[0].ScheduledAt || moved.ScheduledAt != destination.Format(time.RFC3339) || moved.EventVersion != 2 {
		t.Fatal(moved, err)
	}
	if _, err = s.RecordOccurrence(owner, req); status.Code(err) != codes.Aborted {
		t.Fatal("stale move", err)
	}
	old, err = s.ListOccurrences(owner, query(oldDay, today, 100))
	if err != nil || len(old.Items) != 0 {
		t.Fatal("old range still contains moved occurrence", old, err)
	}
	futureQuery := query(today.AddDate(0, 0, 2), today.AddDate(0, 0, 3), 1)
	first, err := s.ListOccurrences(owner, futureQuery)
	if err != nil || !first.HasMore {
		t.Fatal(first, err)
	}
	futureQuery.Cursor = first.NextCursor
	second, err := s.ListOccurrences(owner, futureQuery)
	if err != nil || len(second.Items) != 1 || second.Items[0].OccurrenceId != id || second.Items[0].Status != "SCHEDULED" {
		t.Fatal("moved candidate pagination", second, err)
	}
	req.ExpectedEventVersion = &moved.EventVersion
	req.RescheduledTo = today.AddDate(0, 0, -1).Format(time.RFC3339)
	if _, err = s.RecordOccurrence(owner, req); status.Code(err) != codes.InvalidArgument {
		t.Fatal("past reschedule", err)
	}
	req.RescheduledTo = destination.Add(time.Hour).Format(time.RFC3339)
	moved, err = s.RecordOccurrence(owner, req)
	if err != nil {
		t.Fatal(err)
	}
	done, err := s.RecordOccurrence(owner, &pb.RecordOccurrenceRequest{OccurrenceId: id, Status: "TAKEN", CompletedAt: time.Now().UTC().Format(time.RFC3339), ExpectedEventVersion: &moved.EventVersion})
	if err != nil || done.Status != "TAKEN" || done.ScheduledAt != req.RescheduledTo {
		t.Fatal("completion lost moved date", done, err)
	}
	req.ExpectedEventVersion = &done.EventVersion
	if _, err = s.RecordOccurrence(owner, req); status.Code(err) != codes.InvalidArgument {
		t.Fatal("completed occurrence moved", err)
	}
	user, _ := Principal(owner)
	count, err := s.DB.Collection("health_item_event_history").CountDocuments(owner, bson.M{"user_id": user, "occurrence_id": id})
	if err != nil || count != 4 {
		t.Fatal("audit count", count, err)
	}
	var event healthEventDoc
	if err = s.DB.Collection("health_item_events").FindOne(owner, bson.M{"user_id": user, "occurrence_id": id}).Decode(&event); err != nil || event.RescheduledTo == nil {
		t.Fatal(event, err)
	}
	// Legacy RESCHEDULED records gain effective pending/missed semantics on read.
	_, err = s.DB.Collection("health_item_events").UpdateOne(owner, bson.M{"_id": event.ID}, bson.M{"$set": bson.M{"status": "RESCHEDULED"}})
	if err != nil {
		t.Fatal(err)
	}
	all, err := s.ListOccurrences(owner, query(today.AddDate(0, 0, 2), today.AddDate(0, 0, 3), 100))
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range all.Items {
		if row.OccurrenceId == id && row.Status != "SCHEDULED" {
			t.Fatal("legacy marker leaked", row)
		}
	}
}
