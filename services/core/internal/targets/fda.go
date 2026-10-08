package targets

// FDADailyValues is a versioned transcription of the FDA's food-component
// Daily Values table retrieved on 2026-10-08. These are general food-label
// benchmarks for a 2,000-kcal reference diet, not personalized requirements.
// Source: https://www.fda.gov/media/99059/download
func FDADailyValues() (ReferenceSet, map[string]ReferenceValue) {
	set := ReferenceSet{
		Key: "FDA_DAILY_VALUES_ADULT", Version: "reviewed-2026-10-08", Basis: "GENERAL_BENCHMARK",
		RequiredNutrients: []string{"ENERGY", "PROTEIN", "CARBOHYDRATE", "FAT", "FIBER"},
		Source: SourceAttribution{
			Name:      "FDA Daily Values for food components (2,000-kcal reference diet)",
			Publisher: "U.S. Food and Drug Administration", Version: "reviewed-2026-10-08",
			InformationURL: "https://www.fda.gov/media/99059/download",
		},
	}
	return set, map[string]ReferenceValue{
		"ENERGY":       {Amount: "2000", Unit: "kcal"},
		"PROTEIN":      {Amount: "50", Unit: "g"},
		"CARBOHYDRATE": {Amount: "275", Unit: "g"},
		"FAT":          {Amount: "78", Unit: "g"},
		"FIBER":        {Amount: "28", Unit: "g"},
	}
}
