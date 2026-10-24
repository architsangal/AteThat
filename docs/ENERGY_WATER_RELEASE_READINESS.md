# Next release — maintenance, workouts, deficit and water

Status: implemented locally; release preparation, not published or deployed. Existing production checkpoint remains v2.8.1. Candidate minor version: v2.9.0, subject to release approval.

## Scope

- Settings has an editable maintenance baseline, separate from daily food targets. A profile estimate can be reviewed and selected; saving is always explicit.
- Maintenance uses the existing source-validated adult Inactive energy equations, not BMR or the user's usual-activity multiplier. It includes ordinary movement; workout entries should contain only additional exercise calories to avoid double counting.
- Male: `753.07 − 10.83 × age + 6.50 × height_cm + 14.10 × weight_kg`.
- Female: `584.90 − 7.01 × age + 5.72 × height_cm + 11.71 × weight_kg`.
- Calculation requires age 19+ and supplied equation-specific sex. Missing height/weight use the approved 175 cm and 70.4375 kg assumptions, labelled explicitly. No age/sex is invented. Manual entry remains available when an estimate is unavailable. Pregnancy, breastfeeding and unusual energy needs are not accounted for by this estimate.
- Estimated daily deficit is `effective maintenance + additional logged workout calories − recorded food calories`. Positive means deficit; negative means surplus. It is an estimate based on records, not measured energy balance. Unlogged meals can overstate a deficit.
- A missing food total, baseline or workout calorie value produces an unavailable result, not invented zero. No logged workouts means zero **additional logged** workout calories. Missing water entries stay missing.
- Dashboard shows deficit breakdown and water intake, optional goal progress and a collapsed seven-day water trend.
- Analysis shares the existing applied range: deficit above Daily rhythm, water immediately below it, health charts afterward. Initial seven-day data loads automatically. Changed dates apply only after Explore. Connected dots preserve gaps and signed surplus values, with exact daily tables and responsive scrollable charts.
- Factual Text Summary now includes daily signed deficit arithmetic and explicit unavailable days.
- USDA paneer review/scientific-decimal normalization and safe actionable review errors are included; see NEXT_RELEASE_FIXES.md.

## Storage and compatibility

No new infrastructure, Redis or collection is required. Existing owner-scoped `wellness_settings` embeds append-only `maintenance_history`; the settings version is shared with water-goal edits. Each revision stores kcal, manual/calculated mode, effective day, creation instant, equation and profile-input/version snapshot. History is capped at 1,000 revisions with an explicit maintenance error. Existing account purge removes this document.

First baseline starts on the current Tracking Day. Later revisions start on the next Tracking Day; today's and earlier results are not retroactively changed. Multiple next-day revisions remain auditable; the highest applicable revision wins. Calculated saves require the exact current estimate and profile version. Water-setting edits preserve history and maintenance edits preserve the water goal.

Existing ACTIVITY records remain compatible and are labelled Workout in the UI. Existing food targets, logged food energy, weight measurements, profile weight and schedules remain separate. Historical calorie balance intentionally changes if its underlying food/workout records are explicitly corrected or removed.

REST: authenticated `GET/PUT /wellness/maintenance`, `GET /wellness/analysis?from_tracking_day=...&to_tracking_day=...`. Range is inclusive and bounded to 366 Tracking Days. Generated gRPC contracts and OpenAPI are updated. Deploy Core before API before frontend; no destructive migration or USDA cache purge is needed.

## Local verification — 2026-10-25

- All 60 frontend regression tests passed; production TypeScript/Vite build passed.
- Full API and Core Go suites passed with the disposable local MongoDB replica set, including exact USDA paneer reproduction, scientific-number bounds/precision and safe error/log redaction.
- Maintenance tests cover signed deficit/surplus arithmetic, unknown calories, deleted workouts, summed water, stale settings, unchanged food targets, next-day effective history and owner isolation.
- Calculated-baseline integration tests passed: labelled height/weight assumptions, valid profile estimate, stale profile rejection, tampered estimate rejection, and independence from the usual-activity preference. Factual-summary integration verifies the same deficit arithmetic.
- OpenAPI validated with 39 operations and 145 local references; protobuf descriptor validation passed.
- Chrome local checks: manual baseline save, next-day revision without changing today's value, water and named-workout logging, automatic Analysis, Dashboard totals matching Analysis, chart placement, 390px mobile layout without page overflow, and light/dark card contrast. Existing System theme and seven selected tracked nutrients were preserved.
- Chrome caught missing API route dispatch; fixed and added authentication/dispatch regression cases for all new routes. No production records were changed.
- Applied a 30-day range with Explore in Chrome and verified shared results. Regenerated Text Summary includes the same `2000 + 150 − 90 = 2060 kcal` local fixture as Dashboard/Analysis. Core and API were restarted on the final local build; frontend preview remains on localhost:5175.

Non-blocking tooling warnings: existing Vite bundle exceeds 500 kB; macOS SDK linker-version warnings; optional Ruby ffi extension warning during contract checking. These do not fail validation. Bundle splitting is a follow-up performance improvement, not a hidden release gate.

## Publication checklist — still pending

1. Review the local diff and approve release/version.
2. Commit and push through the protected development/release PR workflow; wait for required checks.
3. Deploy Core, API, then frontend and run authenticated production smoke checks using the designated test account.
4. Tag the verified production commit and publish GitHub release notes. Do not describe an unpublished candidate as released.
5. Retain v2.8.1 as the previous code checkpoint. A code tag is not a database backup; rollback must preserve newly written maintenance history.
