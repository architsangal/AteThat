package server

import (
	"context"
	"sort"
	"time"

	pb "github.com/architsangal/AteThat/services/core/internal/gen/atethat/v1"
	"go.mongodb.org/mongo-driver/bson"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func parseDayRange(fromText, toText string) (time.Time, time.Time, error) {
	from, errA := time.Parse("2006-01-02", fromText)
	to, errB := time.Parse("2006-01-02", toText)
	if errA != nil || errB != nil || from.Format("2006-01-02") != fromText || to.Format("2006-01-02") != toText || to.Before(from) || to.Sub(from) > 366*24*time.Hour {
		return time.Time{}, time.Time{}, invalid("tracking-day range must be at most 366 days")
	}
	return from, to, nil
}

type liveDay struct {
	Groups     []foodLogItemDoc
	GroupCount int
	ItemCount  int
	Timezone   string
	Reset      string
}

func (s *Server) NutritionHistory(ctx context.Context, req *pb.NutritionHistoryRequest) (*pb.NutritionHistoryPage, error) {
	userID, err := Principal(ctx)
	if err != nil {
		return nil, err
	}
	if _, _, err := parseDayRange(req.FromTrackingDay, req.ToTrackingDay); err != nil {
		return nil, err
	}
	if req.Limit == 0 || req.Limit > 50 {
		return nil, invalid("invalid page limit")
	}
	upper := req.ToTrackingDay
	if req.Cursor != "" {
		parsed, err := time.Parse("2006-01-02", req.Cursor)
		if err != nil || parsed.Format("2006-01-02") != req.Cursor || req.Cursor <= req.FromTrackingDay || req.Cursor > req.ToTrackingDay {
			return nil, invalid("invalid cursor")
		}
		upper = parsed.AddDate(0, 0, -1).Format("2006-01-02")
	}
	match := bson.M{"user_id": userID, "tracking_day": bson.M{"$gte": req.FromTrackingDay, "$lte": upper}}
	current := map[string]*liveDay{}
	groups, err := s.DB.Collection("food_log_groups").Find(ctx, bson.M{"user_id": userID, "status": "CONFIRMED",
		"tracking_day": bson.M{"$gte": req.FromTrackingDay, "$lte": upper}})
	if err != nil {
		return nil, internal()
	}
	defer groups.Close(ctx)
	for groups.Next(ctx) {
		var group foodLogGroupDoc
		if groups.Decode(&group) != nil {
			return nil, internal()
		}
		day := current[group.TrackingDay]
		if day == nil {
			day = &liveDay{Timezone: group.TimezoneAtLogging, Reset: group.DailyResetTimeAtLog}
			current[group.TrackingDay] = day
		}
		day.Groups = append(day.Groups, foodLogItemDoc{FinalNutrition: group.Aggregate})
		day.GroupCount++
		day.ItemCount += group.ItemCount
	}
	if groups.Err() != nil {
		return nil, internal()
	}
	summaries := map[string]dailyNutritionSummaryDoc{}
	rows, err := s.DB.Collection("daily_nutrition_summaries").Find(ctx, match)
	if err != nil {
		return nil, internal()
	}
	defer rows.Close(ctx)
	for rows.Next(ctx) {
		var summary dailyNutritionSummaryDoc
		if rows.Decode(&summary) != nil {
			return nil, internal()
		}
		summaries[summary.TrackingDay] = summary
	}
	if rows.Err() != nil {
		return nil, internal()
	}
	days := make([]string, 0, len(current)+len(summaries))
	for day := range current {
		days = append(days, day)
	}
	for day := range summaries {
		if _, exists := current[day]; exists {
			return nil, status.Error(codes.Internal, "compacted day still has detail")
		}
		days = append(days, day)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(days)))
	page := &pb.NutritionHistoryPage{Items: []*pb.NutritionDaySummary{}}
	for _, day := range days {
		if len(page.Items) == int(req.Limit) {
			page.HasMore = true
			break
		}
		if summary, ok := summaries[day]; ok {
			page.Items = append(page.Items, &pb.NutritionDaySummary{TrackingDay: day, Timezone: summary.TimezoneAtSummary,
				DailyResetTime: summary.DailyResetTime, Totals: nutrientPB(summary.Aggregate),
				ConfirmedGroupCount: uint32(summary.ConfirmedGroupCount), ConfirmedItemCount: uint32(summary.ConfirmedItemCount), DetailCompacted: true})
		} else {
			live := current[day]
			totals, err := aggregateNutrition(live.Groups)
			if err != nil {
				return nil, internal()
			}
			page.Items = append(page.Items, &pb.NutritionDaySummary{TrackingDay: day, Timezone: live.Timezone, DailyResetTime: live.Reset,
				Totals: nutrientPB(totals), ConfirmedGroupCount: uint32(live.GroupCount), ConfirmedItemCount: uint32(live.ItemCount)})
		}
	}
	if page.HasMore {
		page.NextCursor = page.Items[len(page.Items)-1].TrackingDay
	}
	return page, nil
}

func summaryBounds(summary dailyNutritionSummaryDoc) (time.Time, time.Time, error) {
	location, err := time.LoadLocation(summary.TimezoneAtSummary)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	date, err := time.Parse("2006-01-02", summary.TrackingDay)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	reset, err := time.Parse("15:04", summary.DailyResetTime)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	start := time.Date(date.Year(), date.Month(), date.Day(), reset.Hour(), reset.Minute(), 0, 0, location)
	next := date.AddDate(0, 0, 1)
	end := time.Date(next.Year(), next.Month(), next.Day(), reset.Hour(), reset.Minute(), 0, 0, location)
	return start.UTC(), end.UTC(), nil
}

func (s *Server) FullAnalysis(ctx context.Context, req *pb.FullAnalysisRequest) (*pb.FullAnalysisResponse, error) {
	userID, err := Principal(ctx)
	if err != nil {
		return nil, err
	}
	from, errA := time.Parse(time.RFC3339Nano, req.From)
	to, errB := time.Parse(time.RFC3339Nano, req.To)
	if errA != nil || errB != nil || !from.Before(to) || to.Sub(from) > 366*24*time.Hour {
		return nil, invalid("analysis range must be at most 366 days")
	}
	from, to = from.UTC(), to.UTC()
	parts := []foodLogItemDoc{}
	groupCount, itemCount, compacted := 0, 0, 0
	groups, err := s.DB.Collection("food_log_groups").Find(ctx, bson.M{"user_id": userID, "status": "CONFIRMED",
		"logged_at_utc": bson.M{"$gte": from, "$lt": to}})
	if err != nil {
		return nil, internal()
	}
	defer groups.Close(ctx)
	for groups.Next(ctx) {
		var group foodLogGroupDoc
		if groups.Decode(&group) != nil {
			return nil, internal()
		}
		parts = append(parts, foodLogItemDoc{FinalNutrition: group.Aggregate})
		groupCount++
		itemCount += group.ItemCount
	}
	if groups.Err() != nil {
		return nil, internal()
	}
	// Compacted summaries are trustworthy only for a completely included
	// Tracking Day. Partial overlap cannot be reconstructed from daily totals.
	summaries, err := s.DB.Collection("daily_nutrition_summaries").Find(ctx, bson.M{"user_id": userID,
		"tracking_day": bson.M{"$gte": from.AddDate(0, 0, -2).Format("2006-01-02"), "$lte": to.AddDate(0, 0, 2).Format("2006-01-02")}})
	if err != nil {
		return nil, internal()
	}
	defer summaries.Close(ctx)
	for summaries.Next(ctx) {
		var summary dailyNutritionSummaryDoc
		if summaries.Decode(&summary) != nil {
			return nil, internal()
		}
		if summary.ConfirmedGroupCount == 0 {
			continue
		}
		start, end := summary.EarliestLoggedAtUTC, summary.LatestLoggedAtUTC
		if start.IsZero() || end.IsZero() {
			// Legacy summaries without observed log bounds use the conservative
			// Tracking Day interval, never a guessed partial result.
			start, end, err = summaryBounds(summary)
			if err != nil {
				return nil, internal()
			}
		} else {
			end = end.Add(time.Nanosecond)
		}
		if !start.Before(to) || !end.After(from) {
			continue
		}
		if start.Before(from) || end.After(to) {
			return nil, status.Error(codes.FailedPrecondition, "partial compacted day cannot be analyzed")
		}
		parts = append(parts, foodLogItemDoc{FinalNutrition: summary.Aggregate})
		groupCount += summary.ConfirmedGroupCount
		itemCount += summary.ConfirmedItemCount
		compacted++
	}
	if summaries.Err() != nil {
		return nil, internal()
	}
	totals := map[string]foodNutrientDoc{}
	if len(parts) > 0 {
		totals, err = aggregateNutrition(parts)
		if err != nil {
			return nil, internal()
		}
	}
	return &pb.FullAnalysisResponse{From: from.Format(time.RFC3339Nano), To: to.Format(time.RFC3339Nano),
		Totals: nutrientPB(totals), ConfirmedGroupCount: uint32(groupCount), ConfirmedItemCount: uint32(itemCount), CompactedDayCount: uint32(compacted)}, nil
}
