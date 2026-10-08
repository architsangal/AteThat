// Package targets defines the source-attributed, versioned review lifecycle for
// nutrition targets. It does not embed or invent dietary reference values.
package targets

import (
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type Status string

const (
	Proposed   Status = "PROPOSED"
	Active     Status = "ACTIVE"
	Rejected   Status = "REJECTED"
	Superseded Status = "SUPERSEDED"
)

var nutrientKeyPattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)
var decimalPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)(\.[0-9]+)?$`)

type SourceAttribution struct {
	Name           string `json:"name" bson:"name"`
	Publisher      string `json:"publisher" bson:"publisher"`
	Version        string `json:"version" bson:"version"`
	InformationURL string `json:"information_url" bson:"information_url"`
}

type ReferenceSet struct {
	Key               string            `json:"key" bson:"key"`
	Version           string            `json:"version" bson:"version"`
	Basis             string            `json:"target_basis" bson:"target_basis"`
	Source            SourceAttribution `json:"source_attribution" bson:"source_attribution"`
	RequiredNutrients []string          `json:"required_nutrients" bson:"required_nutrients"`
}

type ReferenceValue struct {
	Amount string `json:"amount" bson:"amount"`
	Unit   string `json:"unit" bson:"unit"`
}

type Target struct {
	ReferenceValue   string `json:"reference_value" bson:"reference_value"`
	ActiveValue      string `json:"active_value" bson:"active_value"`
	Unit             string `json:"unit" bson:"unit"`
	IsManualOverride bool   `json:"is_manual_override" bson:"is_manual_override"`
	OverrideReason   string `json:"override_reason,omitempty" bson:"override_reason,omitempty"`
}

type Version struct {
	Status                   Status            `json:"status" bson:"status"`
	Revision                 int64             `json:"revision" bson:"revision"`
	ProfileVersion           int64             `json:"profile_version" bson:"profile_version"`
	ReferenceSet             ReferenceSet      `json:"reference_set" bson:"reference_set"`
	TargetBasis              string            `json:"target_basis" bson:"target_basis"`
	CalculationVersion       string            `json:"calculation_version" bson:"calculation_version"`
	EffectiveFromTrackingDay string            `json:"effective_from_tracking_day" bson:"effective_from_tracking_day"`
	Targets                  map[string]Target `json:"targets" bson:"targets"`
}

type Override struct {
	ActiveValue    string `json:"active_value"`
	OverrideReason string `json:"override_reason"`
}

func validDate(day string) bool {
	parsed, err := time.Parse("2006-01-02", day)
	return err == nil && parsed.Format("2006-01-02") == day
}

func validAmount(amount string) bool {
	if len(amount) > 32 || !decimalPattern.MatchString(amount) {
		return false
	}
	value, ok := new(big.Rat).SetString(amount)
	return ok && value.Sign() >= 0
}

func validateSource(source SourceAttribution) error {
	if source.Name == "" || source.Publisher == "" || source.Version == "" {
		return errors.New("reference source attribution is incomplete")
	}
	parsed, err := url.Parse(source.InformationURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return errors.New("reference source needs an HTTPS information URL")
	}
	return nil
}

// NewProposal requires values from an independently reviewed reference set.
// It deliberately has no fallback numbers.
func NewProposal(set ReferenceSet, profileVersion, revision int64, calculationVersion, effectiveDay string, values map[string]ReferenceValue) (Version, error) {
	if set.Key == "" || set.Version == "" || set.Basis == "" || profileVersion < 1 || revision < 1 || calculationVersion == "" || !validDate(effectiveDay) {
		return Version{}, errors.New("invalid proposal metadata")
	}
	if err := validateSource(set.Source); err != nil {
		return Version{}, err
	}
	if len(values) == 0 || len(set.RequiredNutrients) == 0 {
		return Version{}, errors.New("a complete proposal needs reviewed reference values")
	}
	for _, key := range set.RequiredNutrients {
		if !nutrientKeyPattern.MatchString(key) {
			return Version{}, fmt.Errorf("invalid required nutrient %q", key)
		}
		if _, ok := values[key]; !ok {
			return Version{}, fmt.Errorf("required nutrient %q is missing", key)
		}
	}
	targets := make(map[string]Target, len(values))
	for key, value := range values {
		if !nutrientKeyPattern.MatchString(key) || !validAmount(value.Amount) || value.Unit == "" {
			return Version{}, fmt.Errorf("invalid reference value for %q", key)
		}
		targets[key] = Target{ReferenceValue: value.Amount, ActiveValue: value.Amount, Unit: value.Unit}
	}
	return Version{Status: Proposed, Revision: revision, ProfileVersion: profileVersion,
		ReferenceSet: set, TargetBasis: set.Basis, CalculationVersion: calculationVersion,
		EffectiveFromTrackingDay: effectiveDay, Targets: targets}, nil
}

// Accept returns a new value so a failed review cannot partially mutate the
// stored proposal. Storage must atomically activate it and supersede its prior
// active version, with an ownership and status guard.
func Accept(proposal Version, effectiveDay string, overrides map[string]Override) (Version, error) {
	if proposal.Status != Proposed || !validDate(effectiveDay) {
		return Version{}, errors.New("only a proposed version can be accepted")
	}
	accepted := proposal
	accepted.Status = Active
	accepted.EffectiveFromTrackingDay = effectiveDay
	accepted.Targets = make(map[string]Target, len(proposal.Targets))
	for key, value := range proposal.Targets {
		accepted.Targets[key] = value
	}
	for key, override := range overrides {
		target, exists := accepted.Targets[key]
		if !exists || !validAmount(override.ActiveValue) || strings.TrimSpace(override.OverrideReason) == "" || len(override.OverrideReason) > 200 {
			return Version{}, fmt.Errorf("invalid override for %q", key)
		}
		target.ActiveValue = override.ActiveValue
		target.IsManualOverride = true
		target.OverrideReason = override.OverrideReason
		accepted.Targets[key] = target
	}
	return accepted, nil
}

func Reject(proposal Version) (Version, error) {
	if proposal.Status != Proposed {
		return Version{}, errors.New("only a proposed version can be rejected")
	}
	proposal.Status = Rejected
	return proposal, nil
}
