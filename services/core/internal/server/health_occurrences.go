package server

import (
	"container/heap"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"sort"
	"strings"
	"time"

	pb "github.com/architsangal/AteThat/services/core/internal/gen/atethat/v1"
	"github.com/architsangal/AteThat/services/core/internal/health"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type occurrenceIdentity struct {
	Item    string `json:"i"`
	Version int64  `json:"v"`
	At      string `json:"t"`
}
type occurrenceCursor struct {
	At   string `json:"t"`
	ID   string `json:"i"`
	From string `json:"f"`
	To   string `json:"e"`
}

func occurrenceBefore(left, right *pb.HealthOccurrence) bool {
	if left.ScheduledAt == right.ScheduledAt {
		return left.OccurrenceId < right.OccurrenceId
	}
	return left.ScheduledAt < right.ScheduledAt
}

// occurrencePageHeap is a max-heap. Its root is the latest candidate retained
// for the requested page, keeping memory bounded to limit + 1 entries.
type occurrencePageHeap []*pb.HealthOccurrence

func (h occurrencePageHeap) Len() int                { return len(h) }
func (h occurrencePageHeap) Less(i, j int) bool      { return occurrenceBefore(h[j], h[i]) }
func (h occurrencePageHeap) Swap(i, j int)           { h[i], h[j] = h[j], h[i] }
func (h *occurrencePageHeap) Push(value interface{}) { *h = append(*h, value.(*pb.HealthOccurrence)) }
func (h *occurrencePageHeap) Pop() interface{} {
	old := *h
	last := len(old) - 1
	value := old[last]
	*h = old[:last]
	return value
}

func (s *Server) occurrenceID(itemID primitive.ObjectID, version int64, at time.Time) string {
	data, _ := json.Marshal(occurrenceIdentity{Item: itemID.Hex(), Version: version, At: at.UTC().Format(time.RFC3339Nano)})
	mac := hmac.New(sha256.New, s.Cfg.InternalSecret)
	mac.Write(data)
	return base64.RawURLEncoding.EncodeToString(data) + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (s *Server) parseOccurrenceID(raw string) (occurrenceIdentity, error) {
	parts := strings.Split(raw, ".")
	if len(parts) != 2 || len(raw) > 512 {
		return occurrenceIdentity{}, invalid("invalid occurrence ID")
	}
	data, errA := base64.RawURLEncoding.DecodeString(parts[0])
	sig, errB := base64.RawURLEncoding.DecodeString(parts[1])
	if errA != nil || errB != nil || len(data) > 256 {
		return occurrenceIdentity{}, invalid("invalid occurrence ID")
	}
	mac := hmac.New(sha256.New, s.Cfg.InternalSecret)
	mac.Write(data)
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return occurrenceIdentity{}, invalid("invalid occurrence ID")
	}
	var identity occurrenceIdentity
	if json.Unmarshal(data, &identity) != nil || identity.Version < 1 {
		return occurrenceIdentity{}, invalid("invalid occurrence ID")
	}
	return identity, nil
}

func (s *Server) occurrenceFromSchedule(item healthItemDoc, schedule healthScheduleDoc, occurrence health.Occurrence) *pb.HealthOccurrence {
	statusValue := "SCHEDULED"
	if occurrence.At.Before(time.Now().UTC()) {
		statusValue = "MISSED"
	}
	return &pb.HealthOccurrence{OccurrenceId: s.occurrenceID(item.ID, schedule.ScheduleVersion, occurrence.At),
		HealthItemId: item.ID.Hex(), ScheduleVersion: uint64(schedule.ScheduleVersion), Name: item.Name, Type: item.Type,
		ScheduledAt: occurrence.At.UTC().Format(time.RFC3339Nano), ScheduledLocalDate: occurrence.LocalDate,
		ScheduledLocalTime: occurrence.LocalTime, Timezone: occurrence.Timezone, Status: statusValue}
}

func scheduleDomain(doc healthScheduleDoc) health.Schedule {
	return health.Schedule{Timezone: doc.Timezone, Frequency: doc.Frequency, Interval: doc.Interval,
		Weekdays: doc.Weekdays, LocalTimes: doc.LocalTimes, EffectiveFromDate: doc.EffectiveFromDate,
		EffectiveToDate: doc.EffectiveToDate, MonthDays: doc.MonthDays}
}

func eventOnOccurrence(item *pb.HealthOccurrence, event healthEventDoc) {
	item.Status = event.Status
	if event.CompletedAt != nil {
		item.CompletedAt = event.CompletedAt.UTC().Format(time.RFC3339Nano)
	}
	if event.RescheduledTo != nil {
		item.RescheduledTo = event.RescheduledTo.UTC().Format(time.RFC3339Nano)
	}
	item.Notes = event.Notes
}

func (s *Server) ListOccurrences(ctx context.Context, req *pb.ListOccurrencesRequest) (*pb.OccurrencePage, error) {
	userID, err := Principal(ctx)
	if err != nil {
		return nil, err
	}
	from, errA := time.Parse(time.RFC3339Nano, req.From)
	to, errB := time.Parse(time.RFC3339Nano, req.To)
	if errA != nil || errB != nil || !from.Before(to) || to.Sub(from) > 32*24*time.Hour || req.Limit == 0 || req.Limit > 100 {
		return nil, invalid("occurrence range must be at most 32 days and limit 1–100")
	}
	var cursor occurrenceCursor
	if req.Cursor != "" {
		decoded, decodeErr := base64.RawURLEncoding.DecodeString(req.Cursor)
		if decodeErr != nil || len(decoded) > 512 || json.Unmarshal(decoded, &cursor) != nil || cursor.From != req.From || cursor.To != req.To {
			return nil, invalid("invalid occurrence cursor")
		}
	}
	pageSize := int(req.Limit)
	selected := occurrencePageHeap{}
	heap.Init(&selected)
	addCandidate := func(candidate *pb.HealthOccurrence) {
		if req.Cursor != "" && (candidate.ScheduledAt < cursor.At || (candidate.ScheduledAt == cursor.At && candidate.OccurrenceId <= cursor.ID)) {
			return
		}
		if selected.Len() < pageSize+1 {
			heap.Push(&selected, candidate)
			return
		}
		if occurrenceBefore(candidate, selected[0]) {
			heap.Pop(&selected)
			heap.Push(&selected, candidate)
		}
	}
	itemsCursor, err := s.DB.Collection("health_items").Find(ctx, bson.M{"user_id": userID})
	if err != nil {
		return nil, internal()
	}
	defer itemsCursor.Close(ctx)
	for itemsCursor.Next(ctx) {
		var item healthItemDoc
		if itemsCursor.Decode(&item) != nil {
			return nil, internal()
		}
		schedules, err := s.DB.Collection("health_item_schedule_versions").Find(ctx,
			bson.M{"user_id": userID, "health_item_id": item.ID})
		if err != nil {
			return nil, internal()
		}
		for schedules.Next(ctx) {
			var schedule healthScheduleDoc
			if schedules.Decode(&schedule) != nil {
				schedules.Close(ctx)
				return nil, internal()
			}
			occurrences, err := health.Expand(scheduleDomain(schedule), from, to)
			if err != nil {
				schedules.Close(ctx)
				return nil, internal()
			}
			for _, occurrence := range occurrences {
				if occurrence.LocalDate < item.StartDate || (item.EndDate != "" && occurrence.LocalDate > item.EndDate) {
					continue
				}
				addCandidate(s.occurrenceFromSchedule(item, schedule, occurrence))
			}
		}
		if schedules.Err() != nil {
			schedules.Close(ctx)
			return nil, internal()
		}
		schedules.Close(ctx)
	}
	if itemsCursor.Err() != nil {
		return nil, internal()
	}
	items := make([]*pb.HealthOccurrence, selected.Len())
	copy(items, selected)
	sort.Slice(items, func(i, j int) bool { return occurrenceBefore(items[i], items[j]) })
	page := &pb.OccurrencePage{Items: items, HasMore: len(items) > pageSize}
	if page.HasMore {
		page.Items = page.Items[:pageSize]
	}
	if len(page.Items) > 0 {
		ids := make([]string, 0, len(page.Items))
		for _, item := range page.Items {
			ids = append(ids, item.OccurrenceId)
		}
		events, findErr := s.DB.Collection("health_item_events").Find(ctx, bson.M{"user_id": userID, "occurrence_id": bson.M{"$in": ids}})
		if findErr != nil {
			return nil, internal()
		}
		defer events.Close(ctx)
		byID := map[string]healthEventDoc{}
		for events.Next(ctx) {
			var event healthEventDoc
			if events.Decode(&event) != nil {
				return nil, internal()
			}
			byID[event.OccurrenceID] = event
		}
		if events.Err() != nil {
			return nil, internal()
		}
		for _, item := range page.Items {
			if event, ok := byID[item.OccurrenceId]; ok {
				eventOnOccurrence(item, event)
			}
		}
	}
	if page.HasMore {
		last := page.Items[len(page.Items)-1]
		data, _ := json.Marshal(occurrenceCursor{At: last.ScheduledAt, ID: last.OccurrenceId, From: req.From, To: req.To})
		page.NextCursor = base64.RawURLEncoding.EncodeToString(data)
	}
	return page, nil
}

func (s *Server) RecordOccurrence(ctx context.Context, req *pb.RecordOccurrenceRequest) (*pb.HealthOccurrence, error) {
	userID, err := Principal(ctx)
	if err != nil {
		return nil, err
	}
	identity, err := s.parseOccurrenceID(req.OccurrenceId)
	if err != nil {
		return nil, err
	}
	itemID, err := primitive.ObjectIDFromHex(identity.Item)
	at, timeErr := time.Parse(time.RFC3339Nano, identity.At)
	if err != nil || timeErr != nil {
		return nil, invalid("invalid occurrence ID")
	}
	var item healthItemDoc
	if s.DB.Collection("health_items").FindOne(ctx, bson.M{"_id": itemID, "user_id": userID}).Decode(&item) != nil {
		return nil, status.Error(codes.NotFound, "health occurrence not found")
	}
	var schedule healthScheduleDoc
	if s.DB.Collection("health_item_schedule_versions").FindOne(ctx,
		bson.M{"user_id": userID, "health_item_id": itemID, "schedule_version": identity.Version}).Decode(&schedule) != nil {
		return nil, status.Error(codes.NotFound, "health occurrence not found")
	}
	occurrences, err := health.Expand(scheduleDomain(schedule), at.Add(-time.Second), at.Add(time.Second))
	if err != nil {
		return nil, invalid("invalid occurrence")
	}
	var found *pb.HealthOccurrence
	for _, occurrence := range occurrences {
		if occurrence.At.Equal(at) && occurrence.LocalDate >= item.StartDate && (item.EndDate == "" || occurrence.LocalDate <= item.EndDate) {
			found = s.occurrenceFromSchedule(item, schedule, occurrence)
			break
		}
	}
	if found == nil || found.OccurrenceId != req.OccurrenceId {
		return nil, status.Error(codes.NotFound, "health occurrence not found")
	}
	if req.Status != "TAKEN" && req.Status != "SKIPPED" && req.Status != "RESCHEDULED" {
		return nil, invalid("invalid occurrence action")
	}
	if len(req.Notes) > 1000 {
		return nil, invalid("notes too long")
	}
	var completed, rescheduled *time.Time
	if req.Status == "TAKEN" {
		value, err := time.Parse(time.RFC3339Nano, req.CompletedAt)
		if err != nil {
			return nil, invalid("completed_at is required for TAKEN")
		}
		value = value.UTC()
		completed = &value
	} else if req.CompletedAt != "" {
		return nil, invalid("completed_at is only for TAKEN")
	}
	if req.Status == "RESCHEDULED" {
		value, err := time.Parse(time.RFC3339Nano, req.RescheduledTo)
		if err != nil || !value.After(at) {
			return nil, invalid("rescheduled_to must be later than scheduled time")
		}
		value = value.UTC()
		rescheduled = &value
	} else if req.RescheduledTo != "" {
		return nil, invalid("rescheduled_to is only for RESCHEDULED")
	}
	now := time.Now().UTC()
	set := bson.M{"status": req.Status, "notes": req.Notes, "updated_at": now,
		"completed_at_utc": completed, "rescheduled_to_utc": rescheduled}
	var event healthEventDoc
	err = s.DB.Collection("health_item_events").FindOneAndUpdate(ctx,
		bson.M{"user_id": userID, "occurrence_id": req.OccurrenceId},
		bson.M{"$set": set, "$setOnInsert": bson.M{"user_id": userID, "health_item_id": itemID,
			"schedule_version": identity.Version, "occurrence_id": req.OccurrenceId, "scheduled_at_utc": at.UTC(),
			"scheduled_local_date": found.ScheduledLocalDate, "scheduled_local_time": found.ScheduledLocalTime,
			"timezone": found.Timezone, "created_at": now}},
		options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)).Decode(&event)
	if err != nil {
		return nil, internal()
	}
	eventOnOccurrence(found, event)
	return found, nil
}
