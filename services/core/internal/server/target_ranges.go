package server

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"strings"
	"time"

	pb "github.com/architsangal/AteThat/services/core/internal/gen/atethat/v1"
	"github.com/architsangal/AteThat/services/core/internal/targets"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const rangeRevision = "adult-amdr-2005-v1"
const rangeSource = "https://www.canada.ca/en/health-canada/services/food-nutrition/healthy-eating/dietary-reference-intakes/tables/reference-values-macronutrients.html"

func decimalTargetValue(value string) (primitive.Decimal128, error) {
	result, err := primitive.ParseDecimal128(value)
	if err != nil {
		return primitive.Decimal128{}, invalid("target exceeds decimal precision")
	}
	return result, nil
}

type targetRangeDoc struct {
	Minimum     string `bson:"minimum"`
	Maximum     string `bson:"maximum"`
	EnergyBasis string `bson:"energy_basis"`
	SourceURL   string `bson:"source_url"`
	Revision    string `bson:"reference_revision"`
}

func (r *targetRangeDoc) pb() *pb.TargetReferenceRange {
	if r == nil {
		return nil
	}
	return &pb.TargetReferenceRange{Minimum: r.Minimum, Maximum: r.Maximum, EnergyBasis: r.EnergyBasis, SourceUrl: r.SourceURL, ReferenceRevision: r.Revision}
}
func rangeDecimal(n *big.Rat) string {
	return strings.TrimRight(strings.TrimRight(n.FloatString(16), "0"), ".")
}

// Adult AMDRs are guidance, not clinical safety limits. Energy and fibre have
// no fabricated min/max; no adult range is applied to an unknown/minor age.
func applyTargetRanges(values map[string]targetValueDoc, previous map[string]targetValueDoc, age uint32) []*pb.CustomTargetWarning {
	warnings := []*pb.CustomTargetWarning{}
	for _, key := range []string{"ENERGY", "PROTEIN", "CARBOHYDRATE", "FAT", "FIBER"} {
		value, ok := values[key]
		if !ok {
			continue
		}
		value.ReferenceRange = nil
		value.OutsideReference = false
		value.AcknowledgedAt = nil
		value.AcknowledgementHash = ""
		spec, hasRange := map[string][3]string{"PROTEIN": {"0.10", "0.35", "4"}, "CARBOHYDRATE": {"0.45", "0.65", "4"}, "FAT": {"0.20", "0.35", "9"}}[key]
		if age >= 19 && hasRange {
			energy, _ := new(big.Rat).SetString(values["ENERGY"].ActiveValue.String())
			bound := func(fraction string) string {
				f, _ := new(big.Rat).SetString(fraction)
				div, _ := new(big.Rat).SetString(spec[2])
				return rangeDecimal(new(big.Rat).Quo(new(big.Rat).Mul(energy, f), div))
			}
			r := &targetRangeDoc{Minimum: bound(spec[0]), Maximum: bound(spec[1]), EnergyBasis: values["ENERGY"].ActiveValue.String(), SourceURL: rangeSource, Revision: rangeRevision}
			amount, _ := new(big.Rat).SetString(value.ActiveValue.String())
			lo, _ := new(big.Rat).SetString(r.Minimum)
			hi, _ := new(big.Rat).SetString(r.Maximum)
			value.ReferenceRange = r
			value.OutsideReference = amount.Cmp(lo) < 0 || amount.Cmp(hi) > 0
			prior := previous[key]
			if value.OutsideReference {
				if prior.OutsideReference && prior.ActiveValue.String() == value.ActiveValue.String() && prior.ReferenceRange != nil && *prior.ReferenceRange == *r && prior.AcknowledgedAt != nil {
					value.AcknowledgedAt = prior.AcknowledgedAt
					value.AcknowledgementHash = prior.AcknowledgementHash
				} else {
					warnings = append(warnings, &pb.CustomTargetWarning{Nutrient: key, SelectedValue: value.ActiveValue.String(), Unit: value.Unit, ReferenceRange: r.pb()})
				}
			}
		}
		values[key] = value
	}
	return warnings
}
func validateTargetChanges(r *pb.SaveCustomTargetsRequest) error {
	if r.ProfileVersion == 0 || len(r.Changes) == 0 || len(r.Changes) > 5 {
		return invalid("profile version and 1–5 changed fields required")
	}
	for key, value := range r.Changes {
		_, known := map[string]bool{"ENERGY": true, "PROTEIN": true, "CARBOHYDRATE": true, "FAT": true, "FIBER": true}[key]
		if !known || value == nil || !targets.ValidSelectedAmount(key, value.ActiveValue) || len(value.OverrideReason) > 200 {
			return invalid("invalid target change")
		}
		// The actual write also validates Decimal128 precision before entering its transaction.
	}
	if r.EnergyProvenance != "" && r.EnergyProvenance != "CUSTOM" && r.EnergyProvenance != "MACRO_DERIVED" {
		return invalid("invalid calorie provenance")
	}
	if r.EnergyProvenance == "MACRO_DERIVED" && r.Changes["ENERGY"] == nil {
		return invalid("macro-derived provenance requires a deliberate calorie change")
	}
	return nil
}
func customReviewHash(user string, r *pb.SaveCustomTargetsRequest, values map[string]targetValueDoc) string {
	// Includes every final value and range; changing calories invalidates related
	// macro confirmations even when those macro inputs were not edited.
	encoded, _ := json.Marshal(struct {
		Domain, User, Revision string
		Active                 string
		Expected, Profile      uint64
		Changes                map[string]*pb.TargetOverride
		Provenance             string
		Values                 map[string]targetValueDoc
	}{
		"atethat-target-review", user, rangeRevision, r.ExpectedActiveId, r.ExpectedRevision, r.ProfileVersion, r.Changes, r.EnergyProvenance, values})
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}
func (s *Server) signTargetReview(hash string, expires int64) string {
	data, _ := json.Marshal(struct {
		Hash    string
		Expires int64
	}{hash, expires})
	mac := hmac.New(sha256.New, s.Cfg.InternalSecret)
	mac.Write(data)
	return base64.RawURLEncoding.EncodeToString(data) + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
func (s *Server) verifyTargetReview(token, hash string) bool {
	pieces := strings.Split(token, ".")
	if len(pieces) != 2 {
		return false
	}
	data, e := base64.RawURLEncoding.DecodeString(pieces[0])
	if e != nil {
		return false
	}
	sig, e := base64.RawURLEncoding.DecodeString(pieces[1])
	if e != nil {
		return false
	}
	mac := hmac.New(sha256.New, s.Cfg.InternalSecret)
	mac.Write(data)
	var body struct {
		Hash    string
		Expires int64
	}
	return hmac.Equal(sig, mac.Sum(nil)) && json.Unmarshal(data, &body) == nil && body.Hash == hash && body.Expires > time.Now().Unix() && body.Expires <= time.Now().Add(11*time.Minute).Unix()
}
func (s *Server) ReviewCustom(ctx context.Context, r *pb.SaveCustomTargetsRequest) (*pb.CustomTargetsReview, error) {
	user, e := Principal(ctx)
	if e != nil {
		return nil, e
	}
	if e := validateTargetChanges(r); e != nil {
		return nil, e
	}
	var profile profileDoc
	if e := s.DB.Collection("user_profiles").FindOne(ctx, bson.M{"user_id": user, "version": int64(r.ProfileVersion)}).Decode(&profile); e != nil {
		return nil, status.Error(codes.Aborted, "profile changed; reload before reviewing")
	}
	var current targetVersionDoc
	e = s.DB.Collection("nutrition_target_versions").FindOne(ctx, bson.M{"user_id": user, "status": "ACTIVE"}).Decode(&current)
	if e != nil && e != mongo.ErrNoDocuments {
		return nil, internal()
	}
	if (e == nil && (current.ID.Hex() != r.ExpectedActiveId || uint64(current.Revision) != r.ExpectedRevision)) || (e == mongo.ErrNoDocuments && (r.ExpectedActiveId != "" || r.ExpectedRevision != 0)) {
		return nil, status.Error(codes.Aborted, "targets changed; reload before reviewing")
	}
	values := map[string]targetValueDoc{}
	for k, v := range current.Targets {
		values[k] = v
	}
	for k, v := range r.Changes {
		parsed, err := decimalTargetValue(v.ActiveValue)
		if err != nil {
			return nil, err
		}
		value := values[k]
		value.ActiveValue = parsed
		value.Unit = map[string]string{"ENERGY": "kcal", "PROTEIN": "g", "CARBOHYDRATE": "g", "FAT": "g", "FIBER": "g"}[k]
		value.IsManualOverride = true
		value.OverrideReason = v.OverrideReason
		value.Provenance = "CUSTOM"
		if k == "ENERGY" && r.EnergyProvenance == "MACRO_DERIVED" {
			value.Provenance = "MACRO_DERIVED"
		}
		values[k] = value
	}
	if len(values) != 5 {
		return nil, invalid("all five targets are required")
	}
	// Match the write's reference metadata so the signed payload is deterministic.
	if e == mongo.ErrNoDocuments {
		_, refs := targets.FDADailyValues()
		for k, v := range values {
			v.ReferenceValue, _ = decimalTargetValue(refs[k].Amount)
			values[k] = v
		}
	}
	inputs, err := planningProfile(profile, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	warnings := applyTargetRanges(values, current.Targets, inputs.Age)
	result := &pb.CustomTargetsReview{Warnings: warnings}
	if len(warnings) > 0 {
		result.AcknowledgementToken = s.signTargetReview(customReviewHash(user.Hex(), r, values), time.Now().Add(10*time.Minute).Unix())
	}
	return result, nil
}
