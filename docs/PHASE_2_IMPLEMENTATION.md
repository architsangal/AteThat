# Phase 2 implementation

Phase 1 is complete. Its source baseline is release `v1.0.0`, commit `a5fa61f4afc403b4b8c38351fb34087e206596a0`.

## Approved sequence

1. Richer nutrition analysis and recipes.
2. Water, weight, and activity logging.
3. Advanced health-item scheduling.
4. Barcode scanning is removed from scope by the owner. No additional Indian-food provider is selected or integrated; existing USDA search and user-owned custom foods/recipes remain available. Local-first USDA catalogue/search and per-user discovery quotas are Phase 3 work; see [Phase 3 backlog](PHASE_3_BACKLOG.md).

Notifications/reminders, offline operation, PWA, and AI are excluded from Phase 2. Database backup/recovery implementation is also excluded; Terms and Conditions must accurately state the current recovery limitation. Existing infrastructure is reused.

All implementation is tested locally before a PR. Feature branches start from `dev`; reviewed changes merge into `dev`, then release PRs promote `dev` to `main`. Do not push or deploy this implementation until the owner requests it.

## Milestone 1: initial local implementation

- Nutrient groups, daily trends, known-day averages, current-target comparison, and daily food/meal contribution drill-downs.
- Analysis follows stored Tracking Days. All daily pages are fetched for the requested bounded range. Missing days and unknown values are excluded from averages and shown as chart gaps. Current target comparisons are explicitly distinguished from historical target versions.
- USDA normalization adds vitamins, minerals, sugar, cholesterol, and fat subtypes with explicit units. Older confirmed records keep their original nutrient snapshots.
- Recipes reuse owned `meals` records with optional `recipe_servings` and `preparation` fields. Existing meals without those fields remain ordinary meals.
- Ingredient selection, quantity/state editing, batch/per-serving nutrition review, saved recipe editing/deletion, and fractional portion review/confirmation.
- Recipe portion logging is scaled in Core with decimal/rational arithmetic and enters the existing snapshot/idempotent confirmation pipeline.
- Custom food forms expose the expanded nutrients. Blank values remain unknown.
- Amino acids and soluble/insoluble fibre are matched only when USDA supplies an explicit matching nutrient name and gram unit. No values or provider IDs are inferred. Missing fields remain unknown; custom values can be supplied explicitly.
- Daily analysis also selects the accepted target revision effective on that Tracking Day, independently of the explicitly labelled current-target comparison.
- Portion review supports ingredient-level nutrient overrides. The original calculated provider values remain available separately in confirmed historical snapshots.
- Saved meals populate an editable food-group draft before confirmation. Quantity edits clear portion-specific overrides and invalidate the previous preview. Ordinary food-log overrides expose the expanded nutrient groups, not just macros.
- Existing user-owned serving definitions (for example a bowl mapped to grams) are retained and verified through the API and browser. The mapping fills the draft quantity; the user still reviews before confirmation.
- Terms page includes current Atlas M0 recovery limitations without a recovery guarantee.

## Milestones 2 and 3: local implementation

- Water, weight, and named activity records with add/edit/remove, version checks, range history, and daily trend charts.
- Water supports ml/L, weight kg/lb, and activity minutes with intensity and optional estimated kcal. Canonical amounts and optional estimates are stored as BSON Decimal128. API decimals are strings; conversions use the Core rational calculator with six-decimal precision.
- Each record snapshots the account timezone, daily reset, and derived Tracking Day. Editing amount/notes without changing the instant preserves that original context. Activity estimates do not silently change nutrition targets.
- Optional personal water goal with version checks; dashboard shows today's water/weight/activity and progress against the owner's chosen water goal. This is not a medical recommendation.
- Daily/weekly/monthly interval schedules, multiple times, selected monthly calendar dates, optional last date, versioned pause/resume, and paginated schedule history (including inactive items).
- Monthly dates absent from a month are skipped. A pause creates no occurrences; an ended pause does not automatically resume. A new active schedule version must be created explicitly.
- Schedule replacements take effect on complete future calendar dates, no earlier than tomorrow in the new schedule timezone, so past and today's events cannot be rewritten. Original ended schedules are not extended by subsequent changes.
- Schedule adherence/history shows taken/missed/skipped/rescheduled/upcoming counts and taken/due percentage, excluding future events. Its bounded date filter is explicitly UTC; event rows show saved local schedule dates/timezones. All occurrence pages are fetched.
- Wellness collections are indexed by owner and included in account deletion.

## Contract additions

`POST /meals` and `PATCH /meals/{id}` accept `recipe_servings` (positive decimal string, optional) and `preparation` (up to 4000 characters). Responses return both fields. An empty yield identifies an ordinary saved meal.

`POST /meals/{id}/preview` accepts optional `servings` (positive decimal string). When supplied, each ingredient quantity is multiplied by `servings / recipe_servings` using the Core decimal calculator. Omission previews the complete saved definition. Servings on a meal without a recipe yield are rejected.

These fields are additive in REST, protobuf, and MongoDB; no destructive migration is needed. Recipe updates use the existing version guard and do not mutate confirmed food logs.

`POST /food-log-groups/preview` also accepts `recipe_yield` and `recipe_portions` together for reviewing an unsaved recipe. Both are positive decimal strings. Ingredient scaling takes place in Core; the browser does not calculate persisted nutrition through binary floating point.

Recipe previews additionally accept `overrides: [{position, nutrients}]`. Values are absolute nutrient amounts for the eaten portion, validated by the existing preview pipeline. Changing an override invalidates the UI confirmation until a new preview is reviewed.

`GET/POST /wellness/records`, `PATCH /wellness/records/{id}`, and `DELETE /wellness/records/{id}?version=...` manage owned records. Lists accept `from_tracking_day`, `to_tracking_day`, optional `kind`, `limit` (1–50), and `cursor`; ranges are bounded to 366 Tracking Days. PATCH carries the full editable definition plus current `version`.

`GET/PUT /wellness/settings` reads/saves `{version, water_target_ml}`. Empty goal clears it; no default target is prescribed.

Health schedules add `effective_to_date` and `recurrence.month_days`, plus `MONTHLY` and `PAUSED` frequencies. `GET /health-items/{id}/schedule-history?limit=20&before_version=...` returns owned versions in descending version order with `next_before_version` (0 when exhausted).

## Local verification

- `npm run build:web`
- `node --test apps/web/src/features/nutrition/model.test.mjs apps/web/src/features/wellness/schedule-model.test.mjs`
- Core/API `go test ./...`, with `ATETHAT_TEST_MONGO_URI` pointing to a local replica set for integration tests.
- `node scripts/phase-two-local-smoke.mjs` exercises the local REST workflow using synthetic records.
- Browser review at `http://localhost:5173`, including narrow viewport, loading/error/empty states, recipe edits and portions, and daily analysis/drill-down.

The local smoke script targets only `localhost:18080` and marks only its synthetic fixture account verified. It never reads production secrets or connects to production.

## Release gate

The owner authorized PR promotion and production deployment on 2026-10-16, superseding the earlier local-only restriction. The automated build, model tests, Core/API tests with local MongoDB, and REST smoke workflow are required before a PR. Production completion must be recorded separately after verification.

Chrome access was restored and the pending visual pass was completed on 2026-10-16. Checks include monthly schedule creation, taken/adherence update, pause/reopen/resume, preserved schedule versions, recipe override invalidation/review/confirmation, saved-meal quantity recalculation and vitamin override, named-serving reuse, wellness note editing, and 390-pixel nutrition/wellness/recipe/schedule layouts without page-level horizontal overflow. The pause/reopen pass found and fixed a null time-list crash; frontend and API JSON regression tests cover it. Owner acceptance and authorized PR/release promotion are still required before production.

See [Phase 2 status and remaining decisions](PHASE_2_STATUS.md) for the scope reconciliation and deferred provider work. Notifications in the original PRD are explicitly excluded by the owner's later decision, along with offline/PWA, AI, and recovery implementation.
