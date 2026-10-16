# Nutrition sources by phase

Reconciled: 2026-10-16. Food composition and target references are separate domains. This records implementation/research status, not a new legal opinion or permission to copy third-party tables.

## Phase 1 — implemented general FDA benchmarks

Core uses `FDA_DAILY_VALUES_ADULT`, version `reviewed-2026-10-08`, basis `GENERAL_BENCHMARK`, attributed to the [FDA table](https://www.fda.gov/media/99059/download). Required comparison values: energy 2,000 kcal, protein 50 g, carbohydrate 275 g, fat 78 g and fibre 28 g.

These are general food-label benchmarks, not calculated personal needs or prescriptions. Reference values and user overrides remain distinguishable. The previous “Phase 6 / calculation disabled” description was stale.

## Phase 2 — expanded recorded nutrition

Expanded nutrient groups use explicit provider metadata/units or user input. Missing/unsupported values stay unknown. Expanded recording does not imply a new personalized reference set or recommended daily value for every nutrient. Historical accepted targets are distinct from current comparisons.

## Phase 3 — catalogue and assisted input

USDA FoodData Central remains the food-composition provider. The proposed catalogue preserves provenance and normalization versions; user edits become private foods or log overrides. AI proposals retain uncertainty and require review, not invented precision or clinical targets. Review current provider reuse/attribution requirements before new storage integrations.

No personalized energy formula or AI medical recommendation engine is approved. Provider choice, processing/privacy and source permissions need review before adding external AI or new datasets.

## Phase 4 and later — research, not approved imports

Historical candidates include the [USDA NAL DRI calculator](https://www.nal.usda.gov/human-nutrition-and-food-safety/dri-calculator), [NIH nutrient recommendations](https://ods.od.nih.gov/healthinformation/nutrientrecommendations.aspx) and [ICMR-NIN publications](https://www.nin.res.in/RDA_Full_Report_2024.html). Public access is not blanket permission to embed underlying third-party tables. Reassess exact rights, population, formulas, unsupported cases, attribution and versions before importing/personalizing. Additional Indian-food providers are a separate decision; barcode is removed.
