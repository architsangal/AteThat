package server

import (
	"context"
	"encoding/base64"
	"errors"
	"strconv"
	"time"

	pb "github.com/architsangal/AteThat/services/core/internal/gen/atethat/v1"
	"github.com/architsangal/AteThat/services/core/internal/targets"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const targetCalculationVersion = "fda-dv-benchmark-v1"

func (s *Server) Calculate(ctx context.Context, req *pb.CalculateTargetRequest) (*pb.TargetVersion, error) {
	userID, err := Principal(ctx)
	if err != nil {
		return nil, err
	}
	if req.ProfileVersion == 0 || req.EffectiveFromTrackingDay == "" {
		return nil, invalid("profile version and effective Tracking Day are required")
	}
	var profile profileDoc
	if err := s.DB.Collection("user_profiles").FindOne(ctx, bson.M{"user_id": userID, "version": int64(req.ProfileVersion)}).Decode(&profile); err != nil {
		return nil, status.Error(codes.Aborted, "profile version changed")
	}
	day, err := s.GetCurrentTrackingDay(ctx, &pb.Empty{})
	if err != nil {
		return nil, err
	}
	if req.EffectiveFromTrackingDay != day.TrackingDay {
		return nil, invalid("the initial benchmark must begin on the current Tracking Day")
	}
	if !adultOnDay(profile.DateOfBirth, day.TrackingDay) {
		return nil, invalid("an adult date of birth is required for this FDA benchmark")
	}
	var reference referenceSetDoc
	if err := s.DB.Collection("nutrition_reference_sets").FindOne(ctx,
		bson.M{"reference_key": "FDA_DAILY_VALUES_ADULT", "status": "ACTIVE"}).Decode(&reference); err != nil {
		return nil, status.Error(codes.Unavailable, "nutrition reference set unavailable")
	}
	values := make(map[string]targets.ReferenceValue, len(reference.Values))
	for key, value := range reference.Values {
		values[key] = targets.ReferenceValue{Amount: value.Amount.String(), Unit: value.Unit}
	}
	set := targets.ReferenceSet{Key: reference.ReferenceKey, Version: reference.ReferenceVersion,
		Basis: reference.TargetBasis, Source: reference.Source, RequiredNutrients: reference.RequiredKeys}

	now := time.Now().UTC()
	session, err := s.Client.StartSession()
	if err != nil {
		return nil, internal()
	}
	defer session.EndSession(ctx)
	var created targetVersionDoc
	_, err = session.WithTransaction(ctx, func(sc mongo.SessionContext) (interface{}, error) {
		var last targetVersionDoc
		err := s.DB.Collection("nutrition_target_versions").FindOne(sc, bson.M{"user_id": userID},
			options.FindOne().SetSort(bson.D{{Key: "revision", Value: -1}})).Decode(&last)
		if err != nil && err != mongo.ErrNoDocuments {
			return nil, err
		}
		revision := int64(1)
		if err == nil {
			revision = last.Revision + 1
		}
		proposal, err := targets.NewProposal(set, int64(req.ProfileVersion), revision,
			targetCalculationVersion, req.EffectiveFromTrackingDay, values)
		if err != nil {
			return nil, err
		}
		created, err = targetDocFromVersion(userID, proposal, now)
		if err != nil {
			return nil, err
		}
		_, err = s.DB.Collection("nutrition_target_versions").UpdateMany(sc,
			bson.M{"user_id": userID, "status": "PROPOSED"},
			bson.M{"$set": bson.M{"status": "REJECTED", "reviewed_at": now}})
		if err != nil {
			return nil, err
		}
		_, err = s.DB.Collection("nutrition_target_versions").InsertOne(sc, created)
		return nil, err
	})
	if err != nil {
		return nil, status.Error(codes.Aborted, "could not create target proposal; please retry")
	}
	return targetPB(created), nil
}

func adultOnDay(dateOfBirth, day string) bool {
	birth, birthErr := time.Parse("2006-01-02", dateOfBirth)
	current, dayErr := time.Parse("2006-01-02", day)
	if birthErr != nil || dayErr != nil || birth.After(current) {
		return false
	}
	return !birth.AddDate(18, 0, 0).After(current)
}

func targetDocFromVersion(userID primitive.ObjectID, version targets.Version, now time.Time) (targetVersionDoc, error) {
	values := make(map[string]targetValueDoc, len(version.Targets))
	for key, value := range version.Targets {
		reference, err := primitive.ParseDecimal128(value.ReferenceValue)
		if err != nil {
			return targetVersionDoc{}, err
		}
		active, err := primitive.ParseDecimal128(value.ActiveValue)
		if err != nil {
			return targetVersionDoc{}, err
		}
		values[key] = targetValueDoc{ReferenceValue: reference, ActiveValue: active, Unit: value.Unit,
			IsManualOverride: value.IsManualOverride, OverrideReason: value.OverrideReason}
	}
	return targetVersionDoc{ID: primitive.NewObjectID(), UserID: userID, Status: string(version.Status),
		Revision: version.Revision, ProfileVersion: version.ProfileVersion,
		ReferenceSet: bson.M{"key": version.ReferenceSet.Key, "version": version.ReferenceSet.Version},
		Source:       version.ReferenceSet.Source, CalculationVersion: version.CalculationVersion,
		TargetBasis: version.TargetBasis, EffectiveFromTrackingDay: version.EffectiveFromTrackingDay,
		Targets: values, CreatedAt: now}, nil
}

func targetPB(doc targetVersionDoc) *pb.TargetVersion {
	values := make(map[string]*pb.TargetValue, len(doc.Targets))
	for key, value := range doc.Targets {
		values[key] = &pb.TargetValue{ReferenceValue: value.ReferenceValue.String(),
			ActiveValue: value.ActiveValue.String(), Unit: value.Unit,
			IsManualOverride: value.IsManualOverride, OverrideReason: value.OverrideReason, Provenance: value.Provenance}
		values[key].ReferenceRange = value.ReferenceRange.pb()
		values[key].OutsideReference = value.OutsideReference
		if value.AcknowledgedAt != nil {
			values[key].AcknowledgedAt = value.AcknowledgedAt.UTC().Format(time.RFC3339Nano)
		}
	}
	result := &pb.TargetVersion{Id: doc.ID.Hex(), Revision: uint64(doc.Revision), Status: doc.Status,
		ProfileVersion: uint64(doc.ProfileVersion), ReferenceSetKey: doc.ReferenceSet["key"].(string),
		ReferenceSetVersion: doc.ReferenceSet["version"].(string),
		SourceAttribution: &pb.TargetSource{Name: doc.Source.Name, Publisher: doc.Source.Publisher,
			Version: doc.Source.Version, InformationUrl: doc.Source.InformationURL},
		CalculationVersion: doc.CalculationVersion, EffectiveFromTrackingDay: doc.EffectiveFromTrackingDay,
		Targets: values, TargetBasis: doc.TargetBasis, CreatedAt: doc.CreatedAt.UTC().Format(time.RFC3339Nano)}
	if doc.ReviewedAt != nil {
		result.ReviewedAt = doc.ReviewedAt.UTC().Format(time.RFC3339Nano)
	}
	if doc.InputSnapshot != nil {
		result.InputSnapshot = doc.InputSnapshot.pb()
	}
	return result
}

func (s *Server) Current(ctx context.Context, _ *pb.Empty) (*pb.TargetSelection, error) {
	userID, err := Principal(ctx)
	if err != nil {
		return nil, err
	}
	result := &pb.TargetSelection{}
	for _, item := range []struct {
		status string
		output **pb.TargetVersion
	}{
		{"ACTIVE", &result.Active}, {"PROPOSED", &result.Proposed},
	} {
		var doc targetVersionDoc
		err := s.DB.Collection("nutrition_target_versions").FindOne(ctx,
			bson.M{"user_id": userID, "status": item.status}).Decode(&doc)
		if err == mongo.ErrNoDocuments {
			continue
		}
		if err != nil {
			return nil, internal()
		}
		*item.output = targetPB(doc)
	}
	return result, nil
}

func domainVersion(doc targetVersionDoc) targets.Version {
	values := make(map[string]targets.Target, len(doc.Targets))
	for key, value := range doc.Targets {
		values[key] = targets.Target{ReferenceValue: value.ReferenceValue.String(),
			ActiveValue: value.ActiveValue.String(), Unit: value.Unit,
			IsManualOverride: value.IsManualOverride, OverrideReason: value.OverrideReason}
	}
	return targets.Version{Status: targets.Status(doc.Status), Revision: doc.Revision,
		ProfileVersion: doc.ProfileVersion, TargetBasis: doc.TargetBasis,
		ReferenceSet: targets.ReferenceSet{Key: doc.ReferenceSet["key"].(string),
			Version: doc.ReferenceSet["version"].(string), Source: doc.Source, Basis: doc.TargetBasis},
		CalculationVersion: doc.CalculationVersion, EffectiveFromTrackingDay: doc.EffectiveFromTrackingDay,
		Targets: values}
}

func (s *Server) Accept(ctx context.Context, req *pb.AcceptTargetRequest) (*pb.TargetVersion, error) {
	userID, err := Principal(ctx)
	if err != nil {
		return nil, err
	}
	id, err := primitive.ObjectIDFromHex(req.VersionId)
	if err != nil {
		return nil, invalid("invalid target version ID")
	}
	var candidate targetVersionDoc
	if e := s.DB.Collection("nutrition_target_versions").FindOne(ctx, bson.M{"_id": id, "user_id": userID}).Decode(&candidate); e == nil && candidate.TargetBasis == "PERSONALIZED_ESTIMATE" {
		return s.acceptCalculatedProposal(ctx, req)
	}
	day, err := s.GetCurrentTrackingDay(ctx, &pb.Empty{})
	if err != nil {
		return nil, err
	}
	if req.EffectiveFromTrackingDay != day.TrackingDay {
		return nil, invalid("the initial benchmark must begin on the current Tracking Day")
	}
	overrides := make(map[string]targets.Override, len(req.Overrides))
	for key, value := range req.Overrides {
		if value == nil || !targets.ValidSelectedAmount(key, value.ActiveValue) {
			return nil, invalid("invalid target amount")
		}
		overrides[key] = targets.Override{ActiveValue: value.ActiveValue, OverrideReason: value.OverrideReason}
	}
	now := time.Now().UTC()
	session, err := s.Client.StartSession()
	if err != nil {
		return nil, internal()
	}
	defer session.EndSession(ctx)
	var accepted targetVersionDoc
	_, err = session.WithTransaction(ctx, func(sc mongo.SessionContext) (interface{}, error) {
		collection := s.DB.Collection("nutrition_target_versions")
		var proposed targetVersionDoc
		if err := collection.FindOne(sc, bson.M{"_id": id, "user_id": userID, "status": "PROPOSED"}).Decode(&proposed); err != nil {
			return nil, err
		}
		version, err := targets.Accept(domainVersion(proposed), req.EffectiveFromTrackingDay, overrides)
		if err != nil {
			return nil, err
		}
		converted, err := targetDocFromVersion(userID, version, proposed.CreatedAt)
		if err != nil {
			return nil, err
		}
		var profile profileDoc
		if e := s.DB.Collection("user_profiles").FindOne(sc, bson.M{"user_id": userID, "version": proposed.ProfileVersion}).Decode(&profile); e != nil {
			return nil, status.Error(codes.Aborted, "Profile changed; prepare a fresh proposal.")
		}
		inputs, e := planningProfile(profile, now)
		if e != nil {
			return nil, e
		}
		for _, warning := range applyTargetRanges(converted.Targets, nil, inputs.Age) {
			if req.Overrides[warning.Nutrient] != nil {
				return nil, status.Error(codes.FailedPrecondition, "Accept the reference set first, then review outside-reference custom targets in Settings.")
			}
		}
		var previous targetVersionDoc
		err = collection.FindOne(sc, bson.M{"user_id": userID, "status": "ACTIVE"}).Decode(&previous)
		if err != nil && err != mongo.ErrNoDocuments {
			return nil, err
		}
		set := bson.M{"status": "ACTIVE", "reviewed_at": now, "effective_from_tracking_day": req.EffectiveFromTrackingDay,
			"targets": converted.Targets}
		if err == nil {
			set["supersedes_version_id"] = previous.ID
			if _, err := collection.UpdateOne(sc, bson.M{"_id": previous.ID, "user_id": userID, "status": "ACTIVE"},
				bson.M{"$set": bson.M{"status": "SUPERSEDED"}}); err != nil {
				return nil, err
			}
		}
		if err := collection.FindOneAndUpdate(sc, bson.M{"_id": id, "user_id": userID, "status": "PROPOSED"},
			bson.M{"$set": set}, options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&accepted); err != nil {
			return nil, err
		}
		return nil, nil
	})
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, status.Error(codes.NotFound, "proposal not found")
		}
		return nil, status.Error(codes.Aborted, "could not activate target proposal")
	}
	return targetPB(accepted), nil
}

func (s *Server) Reject(ctx context.Context, req *pb.TargetVersionRequest) (*pb.Empty, error) {
	userID, err := Principal(ctx)
	if err != nil {
		return nil, err
	}
	id, err := primitive.ObjectIDFromHex(req.VersionId)
	if err != nil {
		return nil, invalid("invalid target version ID")
	}
	result, err := s.DB.Collection("nutrition_target_versions").UpdateOne(ctx,
		bson.M{"_id": id, "user_id": userID, "status": "PROPOSED"},
		bson.M{"$set": bson.M{"status": "REJECTED", "reviewed_at": time.Now().UTC()}})
	if err != nil {
		return nil, internal()
	}
	if result.ModifiedCount != 1 {
		return nil, status.Error(codes.NotFound, "proposal not found")
	}
	return &pb.Empty{}, nil
}

func (s *Server) History(ctx context.Context, req *pb.TargetHistoryRequest) (*pb.TargetHistoryPage, error) {
	userID, err := Principal(ctx)
	if err != nil {
		return nil, err
	}
	limit := int64(req.Limit)
	if limit == 0 {
		limit = 20
	}
	if limit < 1 || limit > 50 {
		return nil, invalid("limit must be 1 through 50")
	}
	filter := bson.M{"user_id": userID}
	if req.Cursor != "" {
		raw, err := base64.RawURLEncoding.DecodeString(req.Cursor)
		if err != nil {
			return nil, invalid("invalid history cursor")
		}
		revision, err := strconv.ParseInt(string(raw), 10, 64)
		if err != nil || revision < 1 {
			return nil, invalid("invalid history cursor")
		}
		filter["revision"] = bson.M{"$lt": revision}
	}
	cursor, err := s.DB.Collection("nutrition_target_versions").Find(ctx, filter,
		options.Find().SetSort(bson.D{{Key: "revision", Value: -1}}).SetLimit(limit+1))
	if err != nil {
		return nil, internal()
	}
	defer cursor.Close(ctx)
	result := &pb.TargetHistoryPage{Items: []*pb.TargetVersion{}}
	for cursor.Next(ctx) {
		var doc targetVersionDoc
		if err := cursor.Decode(&doc); err != nil {
			return nil, internal()
		}
		result.Items = append(result.Items, targetPB(doc))
	}
	if err := cursor.Err(); err != nil {
		return nil, internal()
	}
	if int64(len(result.Items)) > limit {
		result.Items = result.Items[:limit]
		result.HasMore = true
		last := result.Items[len(result.Items)-1].Revision
		result.NextCursor = base64.RawURLEncoding.EncodeToString([]byte(strconv.FormatUint(last, 10)))
	}
	return result, nil
}
