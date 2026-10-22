package server

import (
	"context"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"time"

	pb "github.com/architsangal/AteThat/services/core/internal/gen/atethat/v1"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Summary is a bounded, owner-authorized projection. It never persists the text,
// calls an external service, or includes free-form notes or identity attributes.
func (s *Server) TextSummary(ctx context.Context, r *pb.TextSummaryRequest) (*pb.TextSummaryResponse, error) {
	if _, err := Principal(ctx); err != nil {
		return nil, err
	}
	from, to, err := parseDayRange(r.FromTrackingDay, r.ToTrackingDay)
	if err != nil || to.Sub(from) > 365*24*time.Hour {
		return nil, invalid("choose at most 366 Tracking Days")
	}
	if r.Mode != "DAY" && r.Mode != "RANGE" {
		return nil, invalid("summary mode must be DAY or RANGE")
	}
	if r.Mode == "DAY" && !from.Equal(to) {
		return nil, invalid("DAY requires a single day")
	}
	if len(r.NutrientFilter) > 80 {
		return nil, invalid("invalid chart filter")
	}
	current, err := s.GetCurrentTrackingDay(ctx, &pb.Empty{})
	if err != nil {
		return nil, err
	}
	generated := time.Now().UTC()
	var out strings.Builder
	partial := false
	line := func(format string, args ...interface{}) { fmt.Fprintf(&out, format+"\n", args...) }
	failed := func(section string) {
		partial = true
		line("%s unavailable or incomplete. This is not a zero or an empty result.", section)
	}
	line("AteThat factual Text Summary — %s to %s", r.FromTrackingDay, r.ToTrackingDay)
	line("Generated %s; mode %s; current timezone %s; current nutrition reset %s.", generated.Format(time.RFC3339), r.Mode, current.Timezone, current.DailyResetTime)
	line("Chart nutrient filter: %s. Text includes all represented nutrients. Missing values are unknown, never zero. No medical interpretation.", r.NutrientFilter)
	line("\nNUTRITION — saved Tracking Day boundaries (not UTC calendar days)")
	var days []*pb.NutritionDaySummary
	cursor := ""
	nutritionOK := true
	userID, _ := Principal(ctx)
	groupCount, countErr := s.DB.Collection("food_log_groups").CountDocuments(ctx, bson.M{"user_id": userID, "status": "CONFIRMED", "tracking_day": bson.M{"$gte": r.FromTrackingDay, "$lte": r.ToTrackingDay}}, options.Count().SetLimit(10001))
	if countErr != nil || groupCount > 10000 {
		nutritionOK = false
	}
	for page := 0; nutritionOK && page < 8; page++ {
		rows, e := s.NutritionHistory(ctx, &pb.NutritionHistoryRequest{FromTrackingDay: r.FromTrackingDay, ToTrackingDay: r.ToTrackingDay, Limit: 50, Cursor: cursor})
		if e != nil {
			nutritionOK = false
			break
		}
		days = append(days, rows.Items...)
		if !rows.HasMore {
			break
		}
		if rows.NextCursor == "" || rows.NextCursor == cursor || page == 7 {
			nutritionOK = false
			break
		}
		cursor = rows.NextCursor
	}
	if !nutritionOK {
		failed("Nutrition")
	}
	foodCoverage, foodErr := s.summaryFoodCoverage(ctx, r.FromTrackingDay, r.ToTrackingDay)
	if foodErr != nil {
		failed("Known-food nutrient subtotals (10,000-food bound; narrow the range)")
	}
	sort.Slice(days, func(i, j int) bool { return days[i].TrackingDay < days[j].TrackingDay })
	versions := []*pb.TargetVersion{}
	cursor = ""
	targetOK := true
	for page := 0; page < 200; page++ {
		rows, e := s.History(ctx, &pb.TargetHistoryRequest{Limit: 50, Cursor: cursor})
		if e != nil {
			targetOK = false
			break
		}
		versions = append(versions, rows.Items...)
		if !rows.HasMore {
			break
		}
		if rows.NextCursor == "" || rows.NextCursor == cursor || page == 199 {
			targetOK = false
			break
		}
		cursor = rows.NextCursor
	}
	if !targetOK {
		failed("Historical targets")
	}
	if targetOK {
		sort.Slice(versions, func(i, j int) bool {
			if versions[i].EffectiveFromTrackingDay == versions[j].EffectiveFromTrackingDay {
				return versions[i].Revision < versions[j].Revision
			}
			return versions[i].EffectiveFromTrackingDay < versions[j].EffectiveFromTrackingDay
		})
		var baseline *pb.TargetVersion
		for _, v := range versions {
			if (v.Status == "ACTIVE" || v.Status == "SUPERSEDED") && v.EffectiveFromTrackingDay < r.FromTrackingDay {
				baseline = v
			}
		}
		line("Accepted target context (historical revisions; water is independent):")
		for _, v := range versions {
			if v.Status != "ACTIVE" && v.Status != "SUPERSEDED" {
				continue
			}
			if v != baseline && (v.EffectiveFromTrackingDay < r.FromTrackingDay || v.EffectiveFromTrackingDay > r.ToTrackingDay) {
				continue
			}
			line("  Revision %d effective %s; %s; reference %s (%s).", v.Revision, v.EffectiveFromTrackingDay, v.TargetBasis, summaryInline(v.SourceAttribution.GetName()), v.ReferenceSetVersion)
			targetCodes := []string{}
			for code := range v.Targets {
				targetCodes = append(targetCodes, code)
			}
			sort.Strings(targetCodes)
			for _, code := range targetCodes {
				value := v.Targets[code]
				origin := value.Provenance
				if origin == "" {
					if value.IsManualOverride {
						origin = "CUSTOM"
					} else {
						origin = "GENERAL_BENCHMARK"
					}
				}
				line("    Selected %s: %s %s (%s); reference point %s %s, not a range or upper limit.", code, value.ActiveValue, value.Unit, origin, value.ReferenceValue, value.Unit)
			}
		}
	}
	totals := map[string][]string{}
	coverage := map[string]int{}
	units := map[string]string{}
	unitConflict := map[string]bool{}
	keys := map[string]bool{}
	for _, day := range days {
		line("%s: %d meals, %d foods; timezone %s, reset %s; compacted detail: %t.", day.TrackingDay, day.ConfirmedGroupCount, day.ConfirmedItemCount, day.Timezone, day.DailyResetTime, day.DetailCompacted)
		codes := make([]string, 0, len(day.Totals))
		for code := range day.Totals {
			codes = append(codes, code)
			keys[code] = true
		}
		sort.Strings(codes)
		var accepted *pb.TargetVersion
		if targetOK {
			for _, v := range versions {
				if (v.Status == "ACTIVE" || v.Status == "SUPERSEDED") && v.EffectiveFromTrackingDay <= day.TrackingDay && (accepted == nil || v.EffectiveFromTrackingDay > accepted.EffectiveFromTrackingDay || (v.EffectiveFromTrackingDay == accepted.EffectiveFromTrackingDay && v.Revision > accepted.Revision)) {
					accepted = v
				}
			}
		}
		for _, code := range codes {
			n := day.Totals[code]
			text := "Unknown (incomplete nutrient coverage)"
			if n != nil && n.Status == "KNOWN" {
				text = n.Amount + " " + n.Unit
				if u, ok := units[code]; ok && u != n.Unit {
					unitConflict[code] = true
				}
				units[code] = n.Unit
				totals[code] = append(totals[code], n.Amount)
				coverage[code]++
			}
			line("  %s: %s", code, text)
			if (n == nil || n.Status != "KNOWN") && !day.DetailCompacted && foodErr == nil {
				items := foodCoverage[day.TrackingDay]
				amounts := []string{}
				unit := ""
				compatible := true
				for _, item := range items {
					if value, ok := item[code]; ok && value.Status == "KNOWN" && value.Amount != nil {
						if unit != "" && unit != value.Unit {
							compatible = false
						}
						unit = value.Unit
						amounts = append(amounts, value.Amount.String())
					}
				}
				if total, ok := summarySum(amounts); ok && compatible {
					line("    Known-food subtotal %s %s; coverage %d/%d foods; not a complete daily total.", total, unit, len(amounts), len(items))
				} else {
					line("    Known-food subtotal unavailable; not zero.")
				}
			}
		}
		if accepted != nil {
			codes = nil
			for code := range accepted.Targets {
				codes = append(codes, code)
			}
			sort.Strings(codes)
			line("  Accepted targets revision %d, effective %s, basis %s:", accepted.Revision, accepted.EffectiveFromTrackingDay, accepted.TargetBasis)
			for _, code := range codes {
				v := accepted.Targets[code]
				line("    %s: %s %s (%s)", code, v.ActiveValue, v.Unit, v.Provenance)
			}
		} else {
			line("  No verified accepted target for this day.")
		}
	}
	if len(days) == 0 && nutritionOK {
		line("No recorded nutrition days in this selection (not zero intake).")
	}
	codes := []string{}
	for k := range keys {
		codes = append(codes, k)
	}
	sort.Strings(codes)
	for _, code := range codes {
		if unitConflict[code] {
			line("%s: incompatible units; aggregate unavailable.", code)
			partial = true
			continue
		}
		total, ok := summarySum(totals[code])
		if !ok {
			line("%s: Unknown.", code)
			continue
		}
		line("%s known-day subtotal: %s %s; coverage %d/%d recorded days. Days with incomplete nutrients excluded; not a complete intake total.", code, total, units[code], coverage[code], len(days))
	}
	line("\nWATER, WEIGHT AND ACTIVITY — saved Tracking Days")
	settings, e := s.GetSettings(ctx, &pb.FoodEmpty{})
	if e != nil {
		failed("Drinking-water goal")
	} else if settings.WaterTargetMl != "" {
		line("Current user-selected drinking-water goal: %s ml/day; not a total-water recommendation or historical goal.", settings.WaterTargetMl)
	} else {
		line("No drinking-water goal selected.")
	}
	records := []*pb.WellnessRecord{}
	cursor = ""
	wellnessOK := true
	for page := 0; page < 200; page++ {
		rows, e := s.ListRecords(ctx, &pb.WellnessListRequest{FromTrackingDay: r.FromTrackingDay, ToTrackingDay: r.ToTrackingDay, Limit: 50, Cursor: cursor})
		if e != nil {
			wellnessOK = false
			break
		}
		records = append(records, rows.Items...)
		if rows.NextCursor == "" {
			break
		}
		if rows.NextCursor == cursor || page == 199 {
			wellnessOK = false
			break
		}
		cursor = rows.NextCursor
	}
	if !wellnessOK {
		failed("Wellness")
	}
	sort.Slice(records, func(i, j int) bool { return records[i].LoggedAt < records[j].LoggedAt })
	weights := []*pb.WellnessRecord{}
	water := []string{}
	for _, row := range records {
		switch row.Kind {
		case "WATER":
			water = append(water, row.Amount)
			line("Water %s: %s %s; Tracking Day %s (%s, reset %s).", row.LoggedAt, row.Amount, row.Unit, row.TrackingDay, row.Timezone, row.DailyResetTime)
		case "WEIGHT":
			weights = append(weights, row)
			line("Weight measurement %s: %s %s (not profile weight).", row.LoggedAt, row.Amount, row.Unit)
		case "ACTIVITY":
			energy := "not recorded"
			if row.EstimatedKcal != "" {
				energy = row.EstimatedKcal + " kcal"
			}
			line("Activity %s: %s, %s %s, %s intensity; estimated expenditure %s.", row.LoggedAt, summaryInline(row.Name), row.Amount, row.Unit, row.Intensity, energy)
		}
	}
	if value, ok := summarySum(water); ok {
		line("Recorded drinking-water subtotal: %s ml (%d records).", value, len(water))
	} else {
		line("Water not recorded in retrieved results.")
	}
	if len(weights) > 0 {
		first, last := weights[0], weights[len(weights)-1]
		a, aOK := new(big.Rat).SetString(first.Amount)
		b, bOK := new(big.Rat).SetString(last.Amount)
		if aOK && bOK {
			line("Weight: %d measurements; first %s kg at %s; latest %s kg at %s; observed change %s kg. Not a daily average.", len(weights), first.Amount, first.LoggedAt, last.Amount, last.LoggedAt, summaryDecimal(new(big.Rat).Sub(b, a)))
		}
	} else {
		line("Weight not recorded in retrieved results.")
	}
	if len(records) == 0 && wellnessOK {
		line("No wellness records; absence is not zero activity or consumption.")
	}
	// Health occurrences are a different domain from nutrition Tracking Days.
	healthFrom, healthTo := from, to.AddDate(0, 0, 1)
	if r.Mode == "DAY" {
		loc, e := time.LoadLocation(current.Timezone)
		if e != nil {
			return nil, internal()
		}
		healthFrom = time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, loc)
		healthTo = healthFrom.AddDate(0, 0, 1)
	}
	line("\nHEALTH SCHEDULE — [%s, %s), %s calendar bounds; independent of nutrition reset.", healthFrom.UTC().Format(time.RFC3339), healthTo.UTC().Format(time.RFC3339), map[bool]string{true: current.Timezone, false: "UTC"}[r.Mode == "DAY"])
	items := map[string]*pb.HealthItem{}
	cursor = ""
	itemsOK := true
	for page := 0; page < 200; page++ {
		rows, e := s.ListItems(ctx, &pb.ListHealthItemsRequest{Limit: 50, Cursor: cursor})
		if e != nil {
			itemsOK = false
			break
		}
		for _, item := range rows.Items {
			items[item.Id] = item
		}
		if !rows.HasMore {
			break
		}
		if rows.NextCursor == "" || rows.NextCursor == cursor || page == 199 {
			itemsOK = false
			break
		}
		cursor = rows.NextCursor
	}
	if !itemsOK {
		failed("Configured health item details")
	}
	seen := map[string]bool{}
	count := 0
	due := 0
	statusCounts := map[string]int{}
	healthOK := true
	for start := healthFrom; start.Before(healthTo); {
		end := start.Add(31 * 24 * time.Hour)
		if end.After(healthTo) {
			end = healthTo
		}
		cursor = ""
		for page := 0; page < 100; page++ {
			rows, e := s.ListOccurrences(ctx, &pb.ListOccurrencesRequest{From: start.UTC().Format(time.RFC3339), To: end.UTC().Format(time.RFC3339), Limit: 100, Cursor: cursor})
			if e != nil {
				healthOK = false
				break
			}
			for _, o := range rows.Items {
				if seen[o.OccurrenceId] {
					continue
				}
				seen[o.OccurrenceId] = true
				if count == 10000 {
					healthOK = false
					break
				}
				count++
				state := summaryHealthStatus(o)
				statusCounts[o.Type+" · "+state]++
				if instant, err := time.Parse(time.RFC3339, o.ScheduledAt); err == nil && !instant.After(generated) {
					due++
				}
				dose := "current configured dose unavailable"
				if o.Type == "TASK" {
					dose = "generic task; no dose"
				} else if item := items[o.HealthItemId]; item != nil {
					dose = "current configured dose: " + summaryInline(item.DosageAmountText) + " " + summaryInline(item.DosageUnitText) + " (not actual dose taken)"
				}
				line("%s %s: %s; %s; scheduled %s (%s %s %s); recorded completion %s; rescheduled to %s; occurrence %s.", o.Type, summaryInline(o.Name), state, dose, o.ScheduledAt, o.ScheduledLocalDate, o.ScheduledLocalTime, o.Timezone, summaryOrUnknown(o.CompletedAt), summaryOrUnknown(o.RescheduledTo), o.OccurrenceId)
				if o.OriginalScheduledAt != "" {
					line("  Originally scheduled %s; this selection uses the new effective scheduled time.", o.OriginalScheduledAt)
				}
			}
			if !healthOK || !rows.HasMore {
				break
			}
			if rows.NextCursor == "" || rows.NextCursor == cursor || page == 99 {
				healthOK = false
				break
			}
			cursor = rows.NextCursor
		}
		if !healthOK {
			break
		}
		start = end
	}
	if !healthOK {
		failed("Health schedule (limit 10,000 occurrences; narrow the range)")
	} else if count == 0 {
		line("No scheduled occurrences in these bounds.")
	}
	line("Retrieved schedule counts: %d occurrences, %d due by generation time. These are recorded statuses, not a clinical adherence assessment.", count, due)
	states := []string{}
	for state := range statusCounts {
		states = append(states, state)
	}
	sort.Strings(states)
	for _, state := range states {
		line("  %s: %d", state, statusCounts[state])
	}
	if partial {
		line("\nPARTIAL SUMMARY: unavailable sections must not be interpreted as empty or zero.")
	}
	return &pb.TextSummaryResponse{Text: out.String(), GeneratedAt: generated.Format(time.RFC3339), Partial: partial, Context: r.Mode + "|" + r.FromTrackingDay + "|" + r.ToTrackingDay + "|" + r.NutrientFilter}, nil
}
func summaryInline(s string) string { return strings.Join(strings.Fields(s), " ") }
func summaryOrUnknown(s string) string {
	if s == "" {
		return "not recorded"
	}
	return s
}
func summaryHealthStatus(o *pb.HealthOccurrence) string {
	if o.Status == "MISSED" {
		if o.Type == "TASK" {
			return "overdue — no completion recorded"
		}
		return "overdue — no intake recorded"
	}
	if o.Status == "TAKEN" && o.Type == "TASK" {
		return "done (recorded)"
	}
	return o.Status
}
func summaryDecimal(r *big.Rat) string {
	return strings.TrimRight(strings.TrimRight(r.FloatString(32), "0"), ".")
}
func summarySum(values []string) (string, bool) {
	if len(values) == 0 {
		return "", false
	}
	sum := new(big.Rat)
	for _, value := range values {
		r, ok := new(big.Rat).SetString(value)
		if !ok {
			return "", false
		}
		sum.Add(sum, r)
	}
	return summaryDecimal(sum), true
}

// Only confirmed, currently active revisions belonging to the principal qualify.
// Do not reconstruct past foods from the mutable provider catalogue.
func (s *Server) summaryFoodCoverage(ctx context.Context, from, to string) (map[string][]map[string]foodNutrientDoc, error) {
	user, err := Principal(ctx)
	if err != nil {
		return nil, err
	}
	pipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: bson.M{"user_id": user, "tracking_day": bson.M{"$gte": from, "$lte": to}}}},
		bson.D{{Key: "$lookup", Value: bson.M{"from": "food_log_groups", "localField": "food_log_group_id", "foreignField": "_id", "as": "group"}}},
		bson.D{{Key: "$match", Value: bson.M{"group": bson.M{"$elemMatch": bson.M{"user_id": user, "status": "CONFIRMED"}}}}},
		bson.D{{Key: "$project", Value: bson.M{"tracking_day": 1, "final_nutrition_snapshot": 1}}},
		bson.D{{Key: "$limit", Value: 10001}},
	}
	rows, err := s.DB.Collection("food_log_items").Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer rows.Close(ctx)
	result := map[string][]map[string]foodNutrientDoc{}
	count := 0
	for rows.Next(ctx) {
		count++
		if count > 10000 {
			return nil, fmt.Errorf("food bound exceeded")
		}
		var item foodLogItemDoc
		if err := rows.Decode(&item); err != nil {
			return nil, err
		}
		result[item.TrackingDay] = append(result[item.TrackingDay], item.FinalNutrition)
	}
	return result, rows.Err()
}
