package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"time"

	"github.com/architsangal/AteThat/services/core/internal/foods"
	pb "github.com/architsangal/AteThat/services/core/internal/gen/atethat/v1"
	"github.com/architsangal/AteThat/services/core/internal/targets"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type planningInputs struct {
	HeightCM        string `bson:"height_cm"`
	WeightKG        string `bson:"weight_kg"`
	Activity        string `bson:"activity"`
	AssumedHeight   bool   `bson:"assumed_height"`
	AssumedWeight   bool   `bson:"assumed_weight"`
	AssumedActivity bool   `bson:"assumed_activity"`
	Sex             string `bson:"sex"`
	Age             uint32 `bson:"age"`
	SavedAt         string `bson:"profile_saved_at"`
}

func (p planningInputs) pb() *pb.TargetInputs {
	return &pb.TargetInputs{HeightCm: p.HeightCM, WeightKg: p.WeightKG, Activity: p.Activity, AssumedHeight: p.AssumedHeight, AssumedWeight: p.AssumedWeight, AssumedActivity: p.AssumedActivity, Sex: p.Sex, Age: p.Age, ProfileSavedAt: p.SavedAt}
}
func planningProfile(p profileDoc, today time.Time) (planningInputs, error) {
	if location, err := time.LoadLocation(p.Timezone); err == nil {
		local := today.In(location)
		today = time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC)
	}
	result := planningInputs{HeightCM: targets.DefaultHeightCM, WeightKG: targets.DefaultWeightKG, Activity: targets.DefaultActivity, AssumedHeight: true, AssumedWeight: true, AssumedActivity: true, Sex: p.Sex, SavedAt: p.UpdatedAt.UTC().Format(time.RFC3339)}
	convert := func(q *quantityDoc, kind string) (string, error) {
		factors := map[string]string{"cm": "1", "m": "100", "in": "2.54"}
		if kind == "weight" {
			factors = map[string]string{"kg": "1", "lb": "0.45359237"}
		}
		factor, ok := factors[q.Unit]
		if !ok {
			return "", invalid("unsupported profile " + kind + " unit")
		}
		return foods.Scale(q.Amount.String(), factor, "1")
	}
	var err error
	if p.Height != nil {
		result.HeightCM, err = convert(p.Height, "height")
		result.AssumedHeight = false
		if err != nil {
			return result, err
		}
	}
	if p.Weight != nil {
		result.WeightKG, err = convert(p.Weight, "weight")
		result.AssumedWeight = false
		if err != nil {
			return result, err
		}
	}
	if p.ActivityLevel != "" {
		result.Activity = p.ActivityLevel
		result.AssumedActivity = false
	}
	if birth, e := time.Parse("2006-01-02", p.DateOfBirth); e == nil && !birth.After(today) {
		age := today.Year() - birth.Year()
		if today.Before(birth.AddDate(age, 0, 0)) {
			age--
		}
		result.Age = uint32(age)
	}
	return result, nil
}

const planningExplanation = "Calculated defaults use your profile, falling back to 175 cm, 70.4375 kg and Moderate activity only when those details are missing. Age and sex are not invented. Activity selects the energy equation and protein rate. Fat uses 27.5% of energy; carbohydrate uses the remaining energy; fibre uses 14 g per 1,000 kcal. Reset requires review and confirmation. Weight logs and water goals are never changed."

func calculatedPlan(user primitive.ObjectID, profile profileDoc, inputs planningInputs) (*pb.TargetPlanning, error) {
	result := &pb.TargetPlanning{Inputs: inputs.pb(), ProfileVersion: uint64(profile.Version), PolicyVersion: targets.PlanningPolicyVersion, RoundingToleranceKcal: targets.MismatchToleranceKcal, Explanation: planningExplanation}
	if profile.DateOfBirth == "" {
		result.MissingInputs = append(result.MissingInputs, "Date of birth")
	}
	if profile.Sex == "" || profile.Sex == "DECLINED" {
		result.MissingInputs = append(result.MissingInputs, "Equation-specific sex (optional; not inferred)")
	}
	if profile.DateOfBirth == "" {
		result.CalculationUnavailableReason = "Add your date of birth to calculate adult defaults. Manual targets and logging remain available."
		return result, nil
	}
	calc, err := targets.CalculateDefaults(inputs.Age, inputs.Sex, inputs.HeightCM, inputs.WeightKG, inputs.Activity)
	if err != nil {
		result.CalculationUnavailableReason = err.Error()
		return result, nil
	}
	for _, value := range calc.Values {
		if _, err := primitive.ParseDecimal128(value); err != nil {
			result.CalculationUnavailableReason = "These inputs exceed the supported calculation precision. Review your profile."
			return result, nil
		}
	}
	result.PersonalizedEnabled = true
	result.CalculatedDefaults, result.EnergyEquation, result.ProteinRate, result.ActivityCategory = calc.Values, calc.Equation, calc.ProteinRate, calc.ActivityCategory
	calculatedValues := map[string]targetValueDoc{}
	for key, value := range calc.Values {
		parsed, _ := decimalTargetValue(value)
		calculatedValues[key] = targetValueDoc{ActiveValue: parsed, Unit: map[string]string{"ENERGY": "kcal", "PROTEIN": "g", "CARBOHYDRATE": "g", "FAT": "g", "FIBER": "g"}[key]}
	}
	result.CalculatedWarnings = applyTargetRanges(calculatedValues, nil, inputs.Age)
	// Recomputed inside the save transaction. Binds the preview to the authenticated
	// owner, current profile, effective age, inputs and server calculation version.
	encoded, err := json.Marshal(struct {
		User, Version string
		Profile       int64
		Inputs        planningInputs
		Values        map[string]string
	}{user.Hex(), targets.CalculatedVersion + ":" + rangeRevision, profile.Version, inputs, calc.Values})
	if err != nil {
		return nil, internal()
	}
	sum := sha256.Sum256(encoded)
	result.ResetPreviewToken = hex.EncodeToString(sum[:])
	return result, nil
}

func (s *Server) Planning(ctx context.Context, r *pb.TargetPlanningRequest) (*pb.TargetPlanning, error) {
	user, err := Principal(ctx)
	if err != nil {
		return nil, err
	}
	var profile profileDoc
	if s.DB.Collection("user_profiles").FindOne(ctx, bson.M{"user_id": user}).Decode(&profile) != nil {
		return nil, internal()
	}
	inputs, err := planningProfile(profile, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	result, err := calculatedPlan(user, profile, inputs)
	if err != nil {
		return nil, err
	}
	if len(r.Macros) > 0 {
		if len(r.Macros) != 3 {
			return nil, invalid("provide exactly protein, carbohydrate and fat")
		}
		result.MacroCalories, err = targets.MacroCalories(r.Macros)
		if err != nil {
			return nil, invalid(err.Error())
		}
	}
	return result, nil
}

func (s *Server) SaveCustom(ctx context.Context, r *pb.SaveCustomTargetsRequest) (*pb.TargetVersion, error) {
	return s.saveSelectedTargets(ctx, r, nil)
}
func (s *Server) ResetCalculated(ctx context.Context, r *pb.ResetCalculatedTargetsRequest) (*pb.TargetVersion, error) {
	if !r.AdultScopeConfirmed || r.PreviewToken == "" {
		return nil, invalid("Review the calculated preview and confirm the adult applicability statement before resetting.")
	}
	return s.saveSelectedTargets(ctx, &pb.SaveCustomTargetsRequest{ExpectedActiveId: r.ExpectedActiveId, ExpectedRevision: r.ExpectedRevision, ProfileVersion: r.ProfileVersion}, r)
}
func (s *Server) saveSelectedTargets(ctx context.Context, r *pb.SaveCustomTargetsRequest, reset *pb.ResetCalculatedTargetsRequest) (*pb.TargetVersion, error) {
	user, err := Principal(ctx)
	if err != nil {
		return nil, err
	}
	if r.ProfileVersion == 0 || (reset == nil && (len(r.Changes) == 0 || len(r.Changes) > 5)) {
		return nil, invalid("profile version and 1–5 changed fields required")
	}
	units := map[string]string{"ENERGY": "kcal", "PROTEIN": "g", "CARBOHYDRATE": "g", "FAT": "g", "FIBER": "g"}
	for key, value := range r.Changes {
		if _, ok := units[key]; !ok || value == nil || !targets.ValidSelectedAmount(key, value.ActiveValue) || len(value.OverrideReason) > 200 {
			return nil, invalid("invalid target change")
		}
		if _, e := primitive.ParseDecimal128(value.ActiveValue); e != nil {
			return nil, invalid("target exceeds decimal precision")
		}
	}
	if r.EnergyProvenance != "" && r.EnergyProvenance != "CUSTOM" && r.EnergyProvenance != "MACRO_DERIVED" {
		return nil, invalid("invalid calorie provenance")
	}
	if r.EnergyProvenance == "MACRO_DERIVED" && r.Changes["ENERGY"] == nil {
		return nil, invalid("macro-derived provenance requires a deliberate calorie change")
	}
	now := time.Now().UTC()
	session, err := s.Client.StartSession()
	if err != nil {
		return nil, internal()
	}
	defer session.EndSession(ctx)
	var saved targetVersionDoc
	_, err = session.WithTransaction(ctx, func(sc mongo.SessionContext) (interface{}, error) {
		var profile profileDoc
		if err := s.DB.Collection("user_profiles").FindOne(sc, bson.M{"user_id": user, "version": int64(r.ProfileVersion)}).Decode(&profile); err != nil {
			return nil, status.Error(codes.Aborted, "profile changed; reload before saving targets")
		}
		if reset != nil {
			inputs, err := planningProfile(profile, now)
			if err != nil {
				return nil, err
			}
			plan, err := calculatedPlan(user, profile, inputs)
			if err != nil {
				return nil, err
			}
			if !plan.PersonalizedEnabled {
				return nil, invalid(plan.CalculationUnavailableReason)
			}
			if plan.ResetPreviewToken != reset.PreviewToken {
				return nil, status.Error(codes.Aborted, "Calculated defaults changed; review a fresh preview before resetting.")
			}
			r.Changes = map[string]*pb.TargetOverride{}
			for key, value := range plan.CalculatedDefaults {
				r.Changes[key] = &pb.TargetOverride{ActiveValue: value}
			}
		}
		col := s.DB.Collection("nutrition_target_versions")
		var current targetVersionDoc
		findErr := col.FindOne(sc, bson.M{"user_id": user, "status": "ACTIVE"}).Decode(&current)
		if findErr != nil && findErr != mongo.ErrNoDocuments {
			return nil, findErr
		}
		if findErr == nil {
			if current.ID.Hex() != r.ExpectedActiveId || uint64(current.Revision) != r.ExpectedRevision {
				return nil, status.Error(codes.Aborted, "targets changed; reload before saving")
			}
		} else if r.ExpectedActiveId != "" || r.ExpectedRevision != 0 {
			return nil, status.Error(codes.Aborted, "target version changed")
		}
		day, e := s.GetCurrentTrackingDay(sc, &pb.Empty{})
		if e != nil {
			return nil, e
		}
		var last targetVersionDoc
		e = col.FindOne(sc, bson.M{"user_id": user}, options.FindOne().SetSort(bson.D{{Key: "revision", Value: -1}})).Decode(&last)
		if e != nil && e != mongo.ErrNoDocuments {
			return nil, e
		}
		saved = current
		saved.ID = primitive.NewObjectID()
		saved.UserID = user
		saved.Status = "ACTIVE"
		saved.Revision = last.Revision + 1
		saved.ProfileVersion = profile.Version
		saved.EffectiveFromTrackingDay = day.TrackingDay
		saved.CreatedAt = now
		saved.ReviewedAt = &now
		saved.Targets = map[string]targetValueDoc{}
		for k, v := range current.Targets {
			saved.Targets[k] = v
		}
		if findErr == mongo.ErrNoDocuments {
			set, values := targets.FDADailyValues()
			saved.ReferenceSet = bson.M{"key": set.Key, "version": set.Version}
			saved.Source = set.Source
			saved.TargetBasis = set.Basis
			saved.CalculationVersion = "manual-targets-v1"
			for key, value := range values {
				if r.Changes[key] == nil {
					return nil, invalid("all five targets are required for an initial manual set")
				}
				ref, _ := primitive.ParseDecimal128(value.Amount)
				saved.Targets[key] = targetValueDoc{ReferenceValue: ref, Unit: value.Unit}
			}
		}
		if reset != nil {
			saved.ReferenceSet = bson.M{"key": "ADULT_CALCULATED_DEFAULTS", "version": targets.CalculatedVersion}
			saved.Source = targets.SourceAttribution{Name: "Adult energy and activity-based planning defaults", Publisher: "National Academies; AteThat activity approximation", Version: targets.CalculatedVersion, InformationURL: targets.EnergySourceURL}
			saved.TargetBasis = "PERSONALIZED_ESTIMATE"
			saved.CalculationVersion = targets.CalculatedVersion
		}
		for key, change := range r.Changes {
			v := saved.Targets[key]
			v.ActiveValue, _ = primitive.ParseDecimal128(change.ActiveValue)
			v.Unit = units[key]
			v.IsManualOverride = true
			v.OverrideReason = change.OverrideReason
			v.Provenance = "CUSTOM"
			if key == "ENERGY" && r.EnergyProvenance == "MACRO_DERIVED" {
				v.Provenance = "MACRO_DERIVED"
			}
			if reset != nil {
				v.Provenance = "CALCULATED_DEFAULT"
				v.IsManualOverride = false
				v.OverrideReason = ""
				v.ReferenceValue = v.ActiveValue
			}
			saved.Targets[key] = v
		}
		if r.EnergyProvenance == "MACRO_DERIVED" {
			values := map[string]string{}
			for _, key := range []string{"PROTEIN", "CARBOHYDRATE", "FAT"} {
				values[key] = saved.Targets[key].ActiveValue.String()
			}
			calories, e := targets.MacroCalories(values)
			calculated, calculatedOK := new(big.Rat).SetString(calories)
			selected, selectedOK := new(big.Rat).SetString(saved.Targets["ENERGY"].ActiveValue.String())
			if e != nil || !calculatedOK || !selectedOK || calculated.Cmp(selected) != 0 {
				return nil, invalid("macro-derived calories must equal current selected macros")
			}
		}
		snapshot, e := planningProfile(profile, now)
		if e != nil {
			return nil, e
		}
		saved.InputSnapshot = &snapshot
		warnings := applyTargetRanges(saved.Targets, current.Targets, snapshot.Age)
		if len(warnings) > 0 {
			reviewHash := customReviewHash(user.Hex(), r, saved.Targets)
			if reset == nil && !s.verifyTargetReview(r.AcknowledgementToken, reviewHash) {
				return nil, status.Error(codes.FailedPrecondition, "Review and confirm the exact outside-reference targets before saving.")
			}
			ack := sha256.Sum256([]byte(r.AcknowledgementToken))
			if reset != nil {
				ack = sha256.Sum256([]byte(reset.PreviewToken))
			}
			for _, warning := range warnings {
				v := saved.Targets[warning.Nutrient]
				acknowledgedAt := now.Truncate(time.Millisecond)
				v.AcknowledgedAt = &acknowledgedAt
				v.AcknowledgementHash = hex.EncodeToString(ack[:])
				saved.Targets[warning.Nutrient] = v
			}
		}
		if findErr == nil {
			saved.SupersedesVersionID = &current.ID
			_, e = col.UpdateOne(sc, bson.M{"_id": current.ID, "user_id": user, "status": "ACTIVE"}, bson.M{"$set": bson.M{"status": "SUPERSEDED"}})
			if e != nil {
				return nil, e
			}
		}
		// A previously prepared proposal must not later replace this reviewed set.
		if _, e = col.UpdateMany(sc, bson.M{"user_id": user, "status": "PROPOSED"}, bson.M{"$set": bson.M{"status": "REJECTED", "reviewed_at": now}}); e != nil {
			return nil, e
		}
		if _, e = col.InsertOne(sc, saved); e != nil {
			return nil, e
		}
		// A profile write creates a transaction conflict with concurrent profile saves.
		if _, e = s.DB.Collection("user_profiles").UpdateOne(sc, bson.M{"user_id": user, "version": profile.Version}, bson.M{"$set": bson.M{"last_target_review_at": now}}); e != nil {
			return nil, e
		}
		return nil, nil
	})
	if err != nil {
		if status.Code(err) != codes.Unknown {
			return nil, err
		}
		return nil, status.Error(codes.Aborted, "could not save target version; reload and retry")
	}
	return targetPB(saved), nil
}
