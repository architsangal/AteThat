package server

import (
	"context"
	"reflect"
	"strings"
	"time"

	pb "github.com/architsangal/AteThat/services/core/internal/gen/atethat/v1"
	"github.com/architsangal/AteThat/services/core/internal/health"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func scheduleFromRequest(userID, itemID primitive.ObjectID, version int64, req *pb.HealthSchedule, now time.Time) (healthScheduleDoc, error) {
	if req == nil {
		return healthScheduleDoc{}, invalid("schedule is required")
	}
	s := health.Schedule{Timezone: req.Timezone, Frequency: req.Frequency, Interval: int(req.Interval),
		Weekdays: req.Weekdays, LocalTimes: req.LocalTimes, EffectiveFromDate: req.EffectiveFromDate, EffectiveToDate: req.EffectiveToDate, MonthDays: req.MonthDays}
	if err := health.Validate(s); err != nil {
		return healthScheduleDoc{}, invalid(err.Error())
	}
	return healthScheduleDoc{ID: primitive.NewObjectID(), UserID: userID, HealthItemID: itemID,
		ScheduleVersion: version, Timezone: s.Timezone, Frequency: s.Frequency, Interval: s.Interval,
		Weekdays: append([]string{}, s.Weekdays...), LocalTimes: append([]string{}, s.LocalTimes...),
		EffectiveFromDate: s.EffectiveFromDate, EffectiveToDate: s.EffectiveToDate, MonthDays: s.MonthDays, CreatedAt: now}, nil
}

func validateHealthItem(req *pb.UpsertHealthItemRequest) error {
	if req.Type != "MEDICATION" && req.Type != "SUPPLEMENT" && req.Type != "TASK" {
		return invalid("type must be medication, supplement or task")
	}
	if req.Type == "TASK" && (req.DosageAmountText != "" || req.DosageUnitText != "") {
		return invalid("tasks do not have dosage fields; use instructions instead")
	}
	if len(strings.TrimSpace(req.Name)) < 2 || len(req.Name) > 120 || len(req.DosageAmountText) > 80 ||
		len(req.DosageUnitText) > 80 || len(req.Instructions) > 500 || len(req.Notes) > 1000 {
		return invalid("invalid health item text")
	}
	start, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil || start.Format("2006-01-02") != req.StartDate {
		return invalid("invalid start date")
	}
	return nil
}

func healthSchedulePB(doc healthScheduleDoc) *pb.HealthSchedule {
	return &pb.HealthSchedule{Timezone: doc.Timezone, Frequency: doc.Frequency, Interval: uint32(doc.Interval),
		Weekdays: doc.Weekdays, LocalTimes: doc.LocalTimes, EffectiveFromDate: doc.EffectiveFromDate,
		Version: uint64(doc.ScheduleVersion), EffectiveToDate: doc.EffectiveToDate, MonthDays: doc.MonthDays}
}

func healthItemPB(doc healthItemDoc, schedule healthScheduleDoc) *pb.HealthItem {
	return &pb.HealthItem{Id: doc.ID.Hex(), Version: uint64(doc.Version), Type: doc.Type,
		Name: doc.Name, DosageAmountText: doc.DosageAmountText, DosageUnitText: doc.DosageUnitText,
		Instructions: doc.Instructions, Notes: doc.Notes, Active: doc.Active, StartDate: doc.StartDate,
		EndDate: doc.EndDate, Schedule: healthSchedulePB(schedule),
		CreatedAt: doc.CreatedAt.UTC().Format(time.RFC3339Nano), UpdatedAt: doc.UpdatedAt.UTC().Format(time.RFC3339Nano)}
}

func (s *Server) loadHealthItem(ctx context.Context, doc healthItemDoc) (*pb.HealthItem, error) {
	var schedule healthScheduleDoc
	err := s.DB.Collection("health_item_schedule_versions").FindOne(ctx,
		bson.M{"user_id": doc.UserID, "health_item_id": doc.ID, "schedule_version": doc.CurrentScheduleVersion}).Decode(&schedule)
	if err != nil {
		return nil, internal()
	}
	return healthItemPB(doc, schedule), nil
}

func (s *Server) CreateItem(ctx context.Context, req *pb.UpsertHealthItemRequest) (*pb.HealthItem, error) {
	userID, err := Principal(ctx)
	if err != nil {
		return nil, err
	}
	if req.Id != "" || req.Version != 0 {
		return nil, invalid("ID and version are not accepted on create")
	}
	if err := validateHealthItem(req); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	id := primitive.NewObjectID()
	schedule, err := scheduleFromRequest(userID, id, 1, req.Schedule, now)
	if err != nil {
		return nil, err
	}
	if schedule.EffectiveFromDate < req.StartDate {
		return nil, invalid("schedule cannot start before item")
	}
	doc := healthItemDoc{ID: id, UserID: userID, Type: req.Type, Name: strings.TrimSpace(req.Name),
		DosageAmountText: req.DosageAmountText, DosageUnitText: req.DosageUnitText,
		Instructions: req.Instructions, Notes: req.Notes, Active: true, StartDate: req.StartDate,
		CurrentScheduleVersion: 1, Version: 1, CreatedAt: now, UpdatedAt: now}
	session, err := s.Client.StartSession()
	if err != nil {
		return nil, internal()
	}
	defer session.EndSession(ctx)
	_, err = session.WithTransaction(ctx, func(sc mongo.SessionContext) (interface{}, error) {
		if _, err := s.DB.Collection("health_items").InsertOne(sc, doc); err != nil {
			return nil, err
		}
		_, err := s.DB.Collection("health_item_schedule_versions").InsertOne(sc, schedule)
		return nil, err
	})
	if err != nil {
		return nil, internal()
	}
	return healthItemPB(doc, schedule), nil
}

func (s *Server) UpdateItem(ctx context.Context, req *pb.UpsertHealthItemRequest) (*pb.HealthItem, error) {
	userID, err := Principal(ctx)
	if err != nil {
		return nil, err
	}
	id, err := primitive.ObjectIDFromHex(req.Id)
	if err != nil || req.Version == 0 {
		return nil, invalid("ID and version are required")
	}
	if err := validateHealthItem(req); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	session, err := s.Client.StartSession()
	if err != nil {
		return nil, internal()
	}
	defer session.EndSession(ctx)
	var updated healthItemDoc
	var currentSchedule healthScheduleDoc
	_, err = session.WithTransaction(ctx, func(sc mongo.SessionContext) (interface{}, error) {
		if err := s.DB.Collection("health_items").FindOne(sc,
			bson.M{"_id": id, "user_id": userID, "version": int64(req.Version), "active": true}).Decode(&updated); err != nil {
			return nil, err
		}
		if err := s.DB.Collection("health_item_schedule_versions").FindOne(sc,
			bson.M{"user_id": userID, "health_item_id": id, "schedule_version": updated.CurrentScheduleVersion}).Decode(&currentSchedule); err != nil {
			return nil, err
		}
		if req.StartDate != updated.StartDate {
			return nil, invalid("item start date is preserved; change the schedule effective date instead")
		}
		set := bson.M{"type": req.Type, "name": strings.TrimSpace(req.Name), "dosage_amount_text": req.DosageAmountText,
			"dosage_unit_text": req.DosageUnitText, "instructions": req.Instructions, "notes": req.Notes,
			"start_date": req.StartDate, "updated_at": now}
		if req.Schedule != nil {
			candidate, err := scheduleFromRequest(userID, id, updated.CurrentScheduleVersion+1, req.Schedule, now)
			if err != nil {
				return nil, err
			}
			changed := !reflect.DeepEqual(candidate.Weekdays, currentSchedule.Weekdays) || !reflect.DeepEqual(candidate.LocalTimes, currentSchedule.LocalTimes) ||
				candidate.Timezone != currentSchedule.Timezone || candidate.Frequency != currentSchedule.Frequency ||
				candidate.Interval != currentSchedule.Interval || candidate.EffectiveFromDate != currentSchedule.EffectiveFromDate || candidate.EffectiveToDate != currentSchedule.EffectiveToDate || !reflect.DeepEqual(candidate.MonthDays, currentSchedule.MonthDays)
			if changed {
				location, _ := time.LoadLocation(candidate.Timezone)
				if candidate.EffectiveFromDate <= now.In(location).Format("2006-01-02") {
					return nil, invalid("schedule changes must start on a future date to preserve today's events")
				}
				if candidate.EffectiveFromDate < req.StartDate || candidate.EffectiveFromDate <= currentSchedule.EffectiveFromDate {
					return nil, invalid("new schedule must begin after the previous version and item start")
				}
				date, _ := time.Parse("2006-01-02", candidate.EffectiveFromDate)
				previousEnd := date.AddDate(0, 0, -1).Format("2006-01-02")
				if currentSchedule.EffectiveToDate != "" && currentSchedule.EffectiveToDate < previousEnd {
					previousEnd = currentSchedule.EffectiveToDate
				}
				if _, err := s.DB.Collection("health_item_schedule_versions").UpdateOne(sc,
					bson.M{"_id": currentSchedule.ID, "user_id": userID}, bson.M{"$set": bson.M{"effective_to_date": previousEnd}}); err != nil {
					return nil, err
				}
				if _, err := s.DB.Collection("health_item_schedule_versions").InsertOne(sc, candidate); err != nil {
					return nil, err
				}
				set["current_schedule_version"] = candidate.ScheduleVersion
				currentSchedule = candidate
			}
		}
		err := s.DB.Collection("health_items").FindOneAndUpdate(sc,
			bson.M{"_id": id, "user_id": userID, "version": int64(req.Version), "active": true},
			bson.M{"$set": set, "$inc": bson.M{"version": 1}},
			options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&updated)
		return nil, err
	})
	if err == mongo.ErrNoDocuments {
		return nil, status.Error(codes.Aborted, "health item version conflict")
	}
	if err != nil {
		if status.Code(err) != codes.Unknown {
			return nil, err
		}
		return nil, status.Error(codes.Aborted, "could not update health item")
	}
	return healthItemPB(updated, currentSchedule), nil
}

func (s *Server) GetItem(ctx context.Context, req *pb.HealthItemRequest) (*pb.HealthItem, error) {
	userID, err := Principal(ctx)
	if err != nil {
		return nil, err
	}
	id, err := primitive.ObjectIDFromHex(req.Id)
	if err != nil {
		return nil, invalid("invalid health item ID")
	}
	var doc healthItemDoc
	err = s.DB.Collection("health_items").FindOne(ctx, bson.M{"_id": id, "user_id": userID}).Decode(&doc)
	if err == mongo.ErrNoDocuments {
		return nil, status.Error(codes.NotFound, "health item not found")
	}
	if err != nil {
		return nil, internal()
	}
	return s.loadHealthItem(ctx, doc)
}

func (s *Server) ListItems(ctx context.Context, req *pb.ListHealthItemsRequest) (*pb.HealthItemPage, error) {
	userID, err := Principal(ctx)
	if err != nil {
		return nil, err
	}
	if req.Limit == 0 || req.Limit > 50 {
		return nil, invalid("invalid page limit")
	}
	filter := bson.M{"user_id": userID}
	if req.ActiveFilter == "true" {
		filter["active"] = true
	} else if req.ActiveFilter == "false" {
		filter["active"] = false
	} else if req.ActiveFilter != "" {
		return nil, invalid("invalid active filter")
	}
	if req.Cursor != "" {
		id, err := primitive.ObjectIDFromHex(req.Cursor)
		if err != nil {
			return nil, invalid("invalid cursor")
		}
		filter["_id"] = bson.M{"$lt": id}
	}
	rows, err := s.DB.Collection("health_items").Find(ctx, filter,
		options.Find().SetSort(bson.D{{Key: "_id", Value: -1}}).SetLimit(int64(req.Limit)+1))
	if err != nil {
		return nil, internal()
	}
	defer rows.Close(ctx)
	result := &pb.HealthItemPage{Items: []*pb.HealthItem{}}
	for rows.Next(ctx) {
		var doc healthItemDoc
		if rows.Decode(&doc) != nil {
			return nil, internal()
		}
		if len(result.Items) == int(req.Limit) {
			result.HasMore = true
			break
		}
		item, err := s.loadHealthItem(ctx, doc)
		if err != nil {
			return nil, err
		}
		result.Items = append(result.Items, item)
	}
	if rows.Err() != nil {
		return nil, internal()
	}
	if result.HasMore {
		result.NextCursor = result.Items[len(result.Items)-1].Id
	}
	return result, nil
}

func (s *Server) DeactivateItem(ctx context.Context, req *pb.DeactivateHealthItemRequest) (*pb.HealthItem, error) {
	userID, err := Principal(ctx)
	if err != nil {
		return nil, err
	}
	id, err := primitive.ObjectIDFromHex(req.Id)
	if err != nil || req.Version == 0 {
		return nil, invalid("ID and version are required")
	}
	end, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil || end.Format("2006-01-02") != req.EndDate {
		return nil, invalid("invalid end date")
	}
	var doc healthItemDoc
	err = s.DB.Collection("health_items").FindOneAndUpdate(ctx,
		bson.M{"_id": id, "user_id": userID, "version": int64(req.Version), "active": true, "start_date": bson.M{"$lte": req.EndDate}},
		bson.M{"$set": bson.M{"active": false, "end_date": req.EndDate, "updated_at": time.Now().UTC()}, "$inc": bson.M{"version": 1}},
		options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&doc)
	if err == mongo.ErrNoDocuments {
		return nil, status.Error(codes.Aborted, "health item version conflict")
	}
	if err != nil {
		return nil, internal()
	}
	return s.loadHealthItem(ctx, doc)
}
