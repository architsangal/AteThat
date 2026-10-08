package targets

import "testing"

func testSet() ReferenceSet {
	return ReferenceSet{Key: "TEST_REFERENCE", Version: "test-1", Basis: "GENERAL_BENCHMARK", RequiredNutrients: []string{"ENERGY", "PROTEIN"}, Source: SourceAttribution{
		Name: "Synthetic test fixture", Publisher: "AteThat tests", Version: "test-1",
		InformationURL: "https://example.com/test-fixture",
	}}
}

func TestProposalReviewAndAttribution(t *testing.T) {
	proposal, err := NewProposal(testSet(), 2, 3, "test-engine-v1", "2026-10-08", map[string]ReferenceValue{
		"ENERGY": {Amount: "2300", Unit: "kcal"}, "PROTEIN": {Amount: "110.50", Unit: "g"},
	})
	if err != nil || proposal.Status != Proposed || proposal.ReferenceSet.Source.Name != "Synthetic test fixture" {
		t.Fatalf("invalid proposal: %+v, %v", proposal, err)
	}
	accepted, err := Accept(proposal, "2026-10-09", map[string]Override{
		"PROTEIN": {ActiveValue: "130.25", OverrideReason: "Personal target"},
	})
	if err != nil || accepted.Status != Active || accepted.Targets["PROTEIN"].ActiveValue != "130.25" ||
		accepted.Targets["PROTEIN"].ReferenceValue != "110.50" || !accepted.Targets["PROTEIN"].IsManualOverride ||
		accepted.Targets["ENERGY"].ActiveValue != "2300" || accepted.ReferenceSet.Source.Name != proposal.ReferenceSet.Source.Name {
		t.Fatalf("review did not preserve target or source: %+v, %v", accepted, err)
	}
	if proposal.Status != Proposed || proposal.Targets["PROTEIN"].ActiveValue != "110.50" {
		t.Fatal("accept mutated the proposal")
	}
	if _, err := Accept(accepted, "2026-10-09", nil); err == nil {
		t.Fatal("active version accepted again")
	}
	rejected, err := Reject(proposal)
	if err != nil || rejected.Status != Rejected || proposal.Status != Proposed {
		t.Fatalf("reject failed: %+v, %v", rejected, err)
	}
}

func TestProposalFailsClosedWithoutReviewedValues(t *testing.T) {
	for _, values := range []map[string]ReferenceValue{
		nil,
		{"ENERGY": {Amount: "-1", Unit: "kcal"}},
		{"ENERGY": {Amount: "NaN", Unit: "kcal"}},
	} {
		if _, err := NewProposal(testSet(), 1, 1, "test", "2026-10-08", values); err == nil {
			t.Fatalf("accepted unreviewed values: %+v", values)
		}
	}
	set := testSet()
	set.Source.InformationURL = "http://example.com/reference"
	if _, err := NewProposal(set, 1, 1, "test", "2026-10-08", map[string]ReferenceValue{"ENERGY": {Amount: "1", Unit: "kcal"}, "PROTEIN": {Amount: "1", Unit: "g"}}); err == nil {
		t.Fatal("accepted insecure attribution URL")
	}
	if _, err := NewProposal(testSet(), 1, 1, "test", "2026-10-08", map[string]ReferenceValue{"ENERGY": {Amount: "1", Unit: "kcal"}}); err == nil {
		t.Fatal("accepted incomplete nutrient set")
	}
}

func TestOverrideValidation(t *testing.T) {
	proposal, err := NewProposal(testSet(), 1, 1, "test", "2026-10-08", map[string]ReferenceValue{"ENERGY": {Amount: "1", Unit: "kcal"}, "PROTEIN": {Amount: "1", Unit: "g"}})
	if err != nil {
		t.Fatal(err)
	}
	for _, overrides := range []map[string]Override{
		{"UNKNOWN": {ActiveValue: "1", OverrideReason: "test"}},
		{"ENERGY": {ActiveValue: "-1", OverrideReason: "test"}},
		{"ENERGY": {ActiveValue: "2"}},
	} {
		if _, err := Accept(proposal, "2026-10-08", overrides); err == nil {
			t.Fatalf("accepted invalid override: %+v", overrides)
		}
	}
}
