package server

import (
	"context"
	"github.com/architsangal/AteThat/services/core/internal/foods"
	pb "github.com/architsangal/AteThat/services/core/internal/gen/atethat/v1"
	"github.com/architsangal/AteThat/services/core/internal/trackingday"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"
	"time"
)

type wellnessDoc struct {
	ID            primitive.ObjectID    `bson:"_id"`
	UserID        primitive.ObjectID    `bson:"user_id"`
	Version       int64                 `bson:"version"`
	Kind          string                `bson:"kind"`
	Amount        primitive.Decimal128  `bson:"amount"`
	Unit          string                `bson:"unit"`
	LoggedAt      time.Time             `bson:"logged_at"`
	TrackingDay   string                `bson:"tracking_day"`
	Timezone      string                `bson:"timezone"`
	Reset         string                `bson:"daily_reset_time"`
	Name          string                `bson:"name"`
	Intensity     string                `bson:"intensity"`
	EstimatedKcal *primitive.Decimal128 `bson:"estimated_kcal,omitempty"`
	Notes         string                `bson:"notes"`
	DeletedAt     *time.Time            `bson:"deleted_at,omitempty"`
}

func wellnessPB(d wellnessDoc) *pb.WellnessRecord {
	kcal := ""
	if d.EstimatedKcal != nil {
		kcal = d.EstimatedKcal.String()
	}
	return &pb.WellnessRecord{Id: d.ID.Hex(), Version: uint64(d.Version), Kind: d.Kind, Amount: d.Amount.String(), Unit: d.Unit, LoggedAt: d.LoggedAt.UTC().Format(time.RFC3339Nano), TrackingDay: d.TrackingDay, Timezone: d.Timezone, DailyResetTime: d.Reset, Name: d.Name, Intensity: d.Intensity, EstimatedKcal: kcal, Notes: d.Notes}
}
func wellnessInput(r *pb.WellnessRecord) (wellnessDoc, error) {
	d := wellnessDoc{Kind: r.Kind, Unit: r.Unit, Name: strings.TrimSpace(r.Name), Intensity: r.Intensity, Notes: strings.TrimSpace(r.Notes)}
	amount := r.Amount
	if _, ok := positiveDecimal(r.Amount); !ok {
		return d, invalid("amount must be a positive decimal")
	}
	if len(r.Notes) > 1000 || len(r.Name) > 120 {
		return d, invalid("record text is too long")
	}
	var err error
	d.LoggedAt, err = time.Parse(time.RFC3339Nano, r.LoggedAt)
	if err != nil || d.LoggedAt.After(time.Now().Add(5*time.Minute)) || d.LoggedAt.Before(time.Now().AddDate(-10, 0, 0)) {
		return d, invalid("record time must include a timezone and be within the last ten years")
	}
	switch r.Kind {
	case "WATER":
		if r.Unit == "L" {
			amount, err = foods.Scale(r.Amount, "1000", "1")
			d.Unit = "ml"
		} else if r.Unit != "ml" {
			return d, invalid("water unit must be ml or L")
		}
	case "WEIGHT":
		if r.Unit == "lb" {
			amount, err = foods.Scale(r.Amount, "0.45359237", "1")
			d.Unit = "kg"
		} else if r.Unit != "kg" {
			return d, invalid("weight unit must be kg or lb")
		}
	case "ACTIVITY":
		if r.Unit != "min" || len(d.Name) < 2 || (r.Intensity != "LIGHT" && r.Intensity != "MODERATE" && r.Intensity != "VIGOROUS") {
			return d, invalid("activity requires a name, minutes, and intensity")
		}
		if r.EstimatedKcal != "" && (len(r.EstimatedKcal) > 32 || !nonnegativeDecimalPattern.MatchString(r.EstimatedKcal)) {
			return d, invalid("invalid estimated expenditure")
		}
	default:
		return d, invalid("unknown record kind")
	}
	if err != nil {
		return d, invalid("could not convert amount")
	}
	if _, ok := positiveDecimal(amount); !ok {
		return d, invalid("amount is below supported precision")
	}
	d.Amount, err = primitive.ParseDecimal128(amount)
	if err != nil {
		return d, invalid("invalid decimal amount")
	}
	if r.EstimatedKcal != "" {
		value, err := primitive.ParseDecimal128(r.EstimatedKcal)
		if err != nil {
			return d, invalid("invalid estimated expenditure")
		}
		d.EstimatedKcal = &value
	}
	if r.Kind != "ACTIVITY" && (r.EstimatedKcal != "" || r.Intensity != "" || r.Name != "") {
		return d, invalid("activity fields require an activity record")
	}
	return d, nil
}
func (s *Server) SaveRecord(ctx context.Context, r *pb.WellnessRecord) (*pb.WellnessRecord, error) {
	user, err := Principal(ctx)
	if err != nil {
		return nil, err
	}
	d, err := wellnessInput(r)
	if err != nil {
		return nil, err
	}
	var profile profileDoc
	var prefs preferencesDoc
	if s.DB.Collection("user_profiles").FindOne(ctx, bson.M{"user_id": user}).Decode(&profile) != nil || s.DB.Collection("user_preferences").FindOne(ctx, bson.M{"user_id": user}).Decode(&prefs) != nil {
		return nil, internal()
	}
	day, err := trackingday.Derive(d.LoggedAt, profile.Timezone, prefs.DailyResetTime)
	if err != nil {
		return nil, internal()
	}
	d.UserID = user
	d.TrackingDay = day.Day
	d.Timezone = profile.Timezone
	d.Reset = prefs.DailyResetTime
	if r.Id == "" {
		if r.Version != 0 {
			return nil, invalid("version is not accepted on create")
		}
		d.ID = primitive.NewObjectID()
		d.Version = 1
		if _, err := s.DB.Collection("wellness_records").InsertOne(ctx, d); err != nil {
			return nil, internal()
		}
	} else {
		d.ID, err = primitive.ObjectIDFromHex(r.Id)
		if err != nil || r.Version == 0 {
			return nil, invalid("record ID and version required")
		}
		var previous wellnessDoc
		filter := bson.M{"_id": d.ID, "user_id": user, "version": int64(r.Version), "kind": r.Kind, "deleted_at": bson.M{"$exists": false}}
		if err := s.DB.Collection("wellness_records").FindOne(ctx, filter).Decode(&previous); err != nil {
			return nil, status.Error(codes.Aborted, "record version conflict")
		}
		// Preserve the saved Tracking Day context when only amount/notes change.
		if d.LoggedAt.Equal(previous.LoggedAt) {
			d.TrackingDay = previous.TrackingDay
			d.Timezone = previous.Timezone
			d.Reset = previous.Reset
		}
		d.Version = previous.Version + 1
		result, err := s.DB.Collection("wellness_records").ReplaceOne(ctx, filter, d)
		if err != nil {
			return nil, internal()
		}
		if result.MatchedCount != 1 {
			return nil, status.Error(codes.Aborted, "record version conflict")
		}
	}
	return wellnessPB(d), nil
}
func (s *Server) ListRecords(ctx context.Context, r *pb.WellnessListRequest) (*pb.WellnessPage, error) {
	user, err := Principal(ctx)
	if err != nil {
		return nil, err
	}
	from, to, err := parseDayRange(r.FromTrackingDay, r.ToTrackingDay)
	if err != nil {
		return nil, err
	}
	if to.Sub(from) > 365*24*time.Hour {
		return nil, invalid("range must be at most 366 inclusive Tracking Days")
	}
	if r.Limit < 1 || r.Limit > 50 {
		return nil, invalid("limit must be 1–50")
	}
	filter := bson.M{"user_id": user, "deleted_at": bson.M{"$exists": false}, "tracking_day": bson.M{"$gte": r.FromTrackingDay, "$lte": r.ToTrackingDay}}
	if r.Kind != "" {
		if r.Kind != "WATER" && r.Kind != "WEIGHT" && r.Kind != "ACTIVITY" {
			return nil, invalid("unknown record kind")
		}
		filter["kind"] = r.Kind
	}
	if r.Cursor != "" {
		id, err := primitive.ObjectIDFromHex(r.Cursor)
		if err != nil {
			return nil, invalid("invalid cursor")
		}
		filter["_id"] = bson.M{"$lt": id}
	}
	rows, err := s.DB.Collection("wellness_records").Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "_id", Value: -1}}).SetLimit(int64(r.Limit)+1))
	if err != nil {
		return nil, internal()
	}
	defer rows.Close(ctx)
	page := &pb.WellnessPage{Items: []*pb.WellnessRecord{}}
	for rows.Next(ctx) {
		var d wellnessDoc
		if rows.Decode(&d) != nil {
			return nil, internal()
		}
		if len(page.Items) == int(r.Limit) {
			page.NextCursor = page.Items[len(page.Items)-1].Id
			break
		}
		page.Items = append(page.Items, wellnessPB(d))
	}
	if rows.Err() != nil {
		return nil, internal()
	}
	return page, nil
}
func (s *Server) DeleteRecord(ctx context.Context, r *pb.WellnessDeleteRequest) (*pb.FoodEmpty, error) {
	user, err := Principal(ctx)
	if err != nil {
		return nil, err
	}
	id, err := primitive.ObjectIDFromHex(r.Id)
	if err != nil || r.Version == 0 {
		return nil, invalid("record ID and version required")
	}
	result, err := s.DB.Collection("wellness_records").UpdateOne(ctx, bson.M{"_id": id, "user_id": user, "version": int64(r.Version), "deleted_at": bson.M{"$exists": false}}, bson.M{"$set": bson.M{"deleted_at": time.Now().UTC()}, "$inc": bson.M{"version": 1}})
	if err != nil {
		return nil, internal()
	}
	if result.MatchedCount != 1 {
		return nil, status.Error(codes.Aborted, "record version conflict")
	}
	return &pb.FoodEmpty{}, nil
}

type wellnessSettingsDoc struct {
	UserID      primitive.ObjectID        `bson:"user_id"`
	WaterTarget *primitive.Decimal128     `bson:"water_target_ml,omitempty"`
	Version     int64                     `bson:"version"`
	Maintenance []*pb.MaintenanceRevision `bson:"maintenance_history,omitempty"`
}

func (s *Server) GetSettings(ctx context.Context, _ *pb.FoodEmpty) (*pb.WellnessSettings, error) {
	user, err := Principal(ctx)
	if err != nil {
		return nil, err
	}
	var d wellnessSettingsDoc
	err = s.DB.Collection("wellness_settings").FindOne(ctx, bson.M{"user_id": user}).Decode(&d)
	if err == mongo.ErrNoDocuments {
		return &pb.WellnessSettings{}, nil
	}
	if err != nil {
		return nil, internal()
	}
	target := ""
	if d.WaterTarget != nil {
		target = d.WaterTarget.String()
	}
	return &pb.WellnessSettings{WaterTargetMl: target, Version: uint64(d.Version)}, nil
}
func (s *Server) SaveSettings(ctx context.Context, r *pb.WellnessSettings) (*pb.WellnessSettings, error) {
	user, err := Principal(ctx)
	if err != nil {
		return nil, err
	}
	if r.WaterTargetMl != "" {
		if _, ok := positiveDecimal(r.WaterTargetMl); !ok {
			return nil, invalid("water target must be positive or empty")
		}
	}
	collection := s.DB.Collection("wellness_settings")
	var target *primitive.Decimal128
	if r.WaterTargetMl != "" {
		value, err := primitive.ParseDecimal128(r.WaterTargetMl)
		if err != nil {
			return nil, invalid("invalid water target")
		}
		target = &value
	}
	if r.Version == 0 {
		_, err = collection.InsertOne(ctx, wellnessSettingsDoc{UserID: user, WaterTarget: target, Version: 1})
	} else {
		var result *mongo.UpdateResult
		result, err = collection.UpdateOne(ctx, bson.M{"user_id": user, "version": int64(r.Version)}, bson.M{"$set": bson.M{"water_target_ml": target}, "$inc": bson.M{"version": 1}})
		if err == nil && result.MatchedCount != 1 {
			return nil, status.Error(codes.Aborted, "settings version conflict")
		}
	}
	if mongo.IsDuplicateKeyError(err) {
		return nil, status.Error(codes.Aborted, "settings version conflict")
	}
	if err != nil {
		return nil, internal()
	}
	return &pb.WellnessSettings{WaterTargetMl: r.WaterTargetMl, Version: r.Version + 1}, nil
}
