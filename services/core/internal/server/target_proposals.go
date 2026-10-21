package server

import (
	"context"
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

func calculatedDocument(user primitive.ObjectID, p profileDoc, plan *pb.TargetPlanning, inputs planningInputs, day string, now time.Time) targetVersionDoc {
	values := map[string]targetValueDoc{}
	for key, value := range plan.CalculatedDefaults {
		n, _ := decimalTargetValue(value)
		values[key] = targetValueDoc{ReferenceValue: n, ActiveValue: n, Unit: map[string]string{"ENERGY": "kcal", "PROTEIN": "g", "CARBOHYDRATE": "g", "FAT": "g", "FIBER": "g"}[key], Provenance: "CALCULATED_DEFAULT"}
	}
	applyTargetRanges(values, nil, inputs.Age)
	return targetVersionDoc{ID: primitive.NewObjectID(), UserID: user, Status: "PROPOSED", ProfileVersion: p.Version,
		InputSnapshot: &inputs, CalculationPreviewToken: plan.ResetPreviewToken,
		ReferenceSet:       bson.M{"key": "ADULT_CALCULATED_DEFAULTS", "version": targets.CalculatedVersion},
		Source:             targets.SourceAttribution{Name: "Adult energy and activity-based planning defaults", Publisher: "National Academies; AteThat activity approximation", Version: targets.CalculatedVersion, InformationURL: targets.EnergySourceURL},
		CalculationVersion: targets.CalculatedVersion, TargetBasis: "PERSONALIZED_ESTIMATE", EffectiveFromTrackingDay: day, Targets: values, CreatedAt: now}
}

// Called automatically by profile/setup UI. Preparing a proposal never changes
// an active target. A declined proposal is not recreated for the same inputs.
func (s *Server) ProposeCalculated(ctx context.Context, _ *pb.Empty) (*pb.TargetVersion, error) {
	user, e := Principal(ctx)
	if e != nil {
		return nil, e
	}
	session, e := s.Client.StartSession()
	if e != nil {
		return nil, internal()
	}
	defer session.EndSession(ctx)
	var proposed targetVersionDoc
	_, e = session.WithTransaction(ctx, func(sc mongo.SessionContext) (interface{}, error) {
		var p profileDoc
		if e := s.DB.Collection("user_profiles").FindOne(sc, bson.M{"user_id": user}).Decode(&p); e != nil {
			return nil, internal()
		}
		now := time.Now().UTC()
		inputs, e := planningProfile(p, now)
		if e != nil {
			return nil, e
		}
		plan, e := calculatedPlan(user, p, inputs)
		if e != nil {
			return nil, e
		}
		if !plan.PersonalizedEnabled {
			return nil, status.Error(codes.FailedPrecondition, plan.CalculationUnavailableReason)
		}
		col := s.DB.Collection("nutrition_target_versions")
		count, e := col.CountDocuments(sc, bson.M{"user_id": user, "status": "ACTIVE"})
		if e != nil {
			return nil, e
		}
		if count > 0 {
			return nil, status.Error(codes.FailedPrecondition, "Existing accepted targets stay active; use explicit reset to replace them.")
		}
		var existing targetVersionDoc
		e = col.FindOne(sc, bson.M{"user_id": user, "calculation_preview_token": plan.ResetPreviewToken, "status": bson.M{"$in": []string{"PROPOSED", "REJECTED"}}}, options.FindOne().SetSort(bson.D{{Key: "revision", Value: -1}})).Decode(&existing)
		if e == nil {
			proposed = existing
			return nil, nil
		}
		if e != mongo.ErrNoDocuments {
			return nil, e
		}
		day, e := s.GetCurrentTrackingDay(sc, &pb.Empty{})
		if e != nil {
			return nil, e
		}
		proposed = calculatedDocument(user, p, plan, inputs, day.TrackingDay, now)
		var last targetVersionDoc
		e = col.FindOne(sc, bson.M{"user_id": user}, options.FindOne().SetSort(bson.D{{Key: "revision", Value: -1}})).Decode(&last)
		if e != nil && e != mongo.ErrNoDocuments {
			return nil, e
		}
		proposed.Revision = last.Revision + 1
		if _, e = col.UpdateMany(sc, bson.M{"user_id": user, "status": "PROPOSED"}, bson.M{"$set": bson.M{"status": "REJECTED", "reviewed_at": now}}); e != nil {
			return nil, e
		}
		if _, e = col.InsertOne(sc, proposed); e != nil {
			return nil, e
		}
		_, e = s.DB.Collection("user_profiles").UpdateOne(sc, bson.M{"user_id": user, "version": p.Version}, bson.M{"$set": bson.M{"last_target_proposal_at": now}})
		return nil, e
	})
	if e != nil {
		if status.Code(e) != codes.Unknown {
			return nil, e
		}
		return nil, status.Error(codes.Aborted, "Profile or proposal changed; retry.")
	}
	return targetPB(proposed), nil
}
func (s *Server) acceptCalculatedProposal(ctx context.Context, r *pb.AcceptTargetRequest) (*pb.TargetVersion, error) {
	user, e := Principal(ctx)
	if e != nil {
		return nil, e
	}
	if !r.AdultScopeConfirmed || len(r.Overrides) > 0 {
		return nil, invalid("Review the calculated proposal and confirm applicability; edit custom values separately.")
	}
	id, e := primitive.ObjectIDFromHex(r.VersionId)
	if e != nil {
		return nil, invalid("invalid proposal ID")
	}
	session, e := s.Client.StartSession()
	if e != nil {
		return nil, internal()
	}
	defer session.EndSession(ctx)
	var accepted targetVersionDoc
	_, e = session.WithTransaction(ctx, func(sc mongo.SessionContext) (interface{}, error) {
		col := s.DB.Collection("nutrition_target_versions")
		if e := col.FindOne(sc, bson.M{"_id": id, "user_id": user, "status": "PROPOSED", "target_basis": "PERSONALIZED_ESTIMATE"}).Decode(&accepted); e != nil {
			return nil, status.Error(codes.Aborted, "Proposal changed; review a fresh proposal.")
		}
		var p profileDoc
		if e := s.DB.Collection("user_profiles").FindOne(sc, bson.M{"user_id": user, "version": accepted.ProfileVersion}).Decode(&p); e != nil {
			return nil, status.Error(codes.Aborted, "Profile changed; review a fresh proposal.")
		}
		now := time.Now().UTC()
		inputs, e := planningProfile(p, now)
		if e != nil {
			return nil, e
		}
		plan, e := calculatedPlan(user, p, inputs)
		if e != nil {
			return nil, e
		}
		if !plan.PersonalizedEnabled || plan.ResetPreviewToken != accepted.CalculationPreviewToken {
			return nil, status.Error(codes.Aborted, "Calculation changed; review a fresh proposal.")
		}
		count, e := col.CountDocuments(sc, bson.M{"user_id": user, "status": "ACTIVE"})
		if e != nil {
			return nil, e
		}
		if count != 0 {
			return nil, status.Error(codes.Aborted, "Targets were already accepted; reload.")
		}
		day, e := s.GetCurrentTrackingDay(sc, &pb.Empty{})
		if e != nil {
			return nil, e
		}
		if r.EffectiveFromTrackingDay != day.TrackingDay {
			return nil, status.Error(codes.Aborted, "Tracking Day changed; review again.")
		}
		for key, v := range accepted.Targets {
			if v.OutsideReference {
				acknowledgedAt := now.Truncate(time.Millisecond)
				v.AcknowledgedAt = &acknowledgedAt
				v.AcknowledgementHash = accepted.CalculationPreviewToken
				accepted.Targets[key] = v
			}
		}
		accepted.Status = "ACTIVE"
		accepted.ReviewedAt = &now
		accepted.EffectiveFromTrackingDay = day.TrackingDay
		if _, e = col.ReplaceOne(sc, bson.M{"_id": id, "user_id": user, "status": "PROPOSED"}, accepted); e != nil {
			return nil, e
		}
		_, e = s.DB.Collection("user_profiles").UpdateOne(sc, bson.M{"user_id": user, "version": p.Version}, bson.M{"$set": bson.M{"last_target_review_at": now}})
		return nil, e
	})
	if e != nil {
		if status.Code(e) != codes.Unknown {
			return nil, e
		}
		return nil, status.Error(codes.Aborted, "Could not accept proposal; reload and retry.")
	}
	return targetPB(accepted), nil
}
