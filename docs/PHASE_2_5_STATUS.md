# Phase 2.5 — local implementation status

Updated 2026-10-22. Release candidate prepared on `fix/phase-2.1-ux`; earlier dated notes below remain historical. The [release report](PHASE_2_5_RELEASE.md) is authoritative for current verification and actual promotion/deployment evidence.

## Release candidate — 2026-10-22

The initial proposal and applicable-range confirmation gaps below are now implemented. Initial calculated proposals are profile-bound and idempotent, require explicit adult applicability/acceptance, remember a decline, and never replace accepted targets automatically. Applicable adult AMDR comparisons use protein 10–35%, carbohydrate 45–65%, and fat 20–35% of the selected calorie basis. There are no fabricated calorie/fibre limits or adult ranges for unknown/minor age. A signed ten-minute confirmation binds owner, exact final values, source revision, calorie basis and optimistic versions. Unchanged confirmed values retain their acknowledgement; changed values/basis require fresh review. Confirmed status remains visible in Settings. API errors distinguish target confirmation from expired authentication.

Verification: 31 frontend regressions; Core integration tests with a disposable Mongo replica set (including signed/forged/expired/replayed confirmations, ownership, stale profile, proposal accept/decline lifecycle, malformed-section partial reporting and 10,001-group bounds); API tests; both local HTTP smoke suites; TypeScript/build and formal contracts. Chrome verified initial proposal acceptance and outside-reference Cancel/Save with preserved independent fields. Further browser and deployment evidence is recorded in the release report. A non-blocking 500-kB bundle warning remains. CI now runs the complete frontend regression set and real Mongo integration tests instead of silently skipping them.

## Earlier release-readiness gaps (superseded by the release candidate)

- Initial profile-based proposal flow: the read-only planning endpoint calculates eligible defaults, and users can explicitly preview/reset them, but the initial proposed-target acceptance flow after profile setup is not yet implemented.
- Applicable source-range warnings/acknowledgements: independent custom saves validate numbers and versions but do not yet implement nutrient-specific out-of-range confirmation, durable acknowledgement bound to exact values/reference revision, and persistent outside-reference labels. Point references must not be converted into invented clinical upper limits.
- Finish remaining acceptance checks, particularly clipboard denial, account switching/session expiry with summary drafts, summary partial-service failures and high-row-count bounds. Existing successful tests are not evidence of full matrix coverage.
- Release promotion/deployment and production smoke checks have not been performed for this local work. Do not label Phase 2.5 fully complete until the feature gaps are completed or explicitly deferred by the owner.

## Phase 1 and Phase 2

Existing authentication, logging, general-benchmark metadata, recipes, nutrition analysis, wellness and scheduling remain the base. Existing accepted target revisions and historical nutrition are preserved. Phase 2.1 food catalogue/correction/task/spacing work in this checkout is not overwritten.

## Phase 2.5 — implemented locally

- Effective inputs use constants only when missing: height 175 cm, weight 70.4375 kg, activity Moderate. Returned provenance flags distinguish assumptions. No age/sex inference or runtime BMI calculation. Unit conversion uses Core decimal arithmetic. Profile weight and wellness measurements stay independent.
- Optional activity and equation-specific sex in Settings/setup; Settings navigation carries an accessible missing-input badge. Profile saving offers target review rather than silent recalculation.
- One target editor in Settings. The Nutrition targets page is revision history. Edits preserve untouched decimal fields; explicit 4/4/9 action changes only calorie draft. Energy must be positive, macro/fibre nonnegative; plain decimal strings at most 32 characters, representable as Decimal128. No scientific notation. Mismatch tolerance is 1 kcal; editing-session notice is dismissible and does not survive reload.
- Target saves are owner-scoped transactions guarded by accepted target ID/revision and profile version. New accepted revisions supersede prior ones; input snapshot and edited-field provenance are retained. Other fields, profile, water goal and logged food energy do not change.
- Versioned calculation-explanation page linked from Dashboard and Settings. Formula-policy limitations are explicit. FDA point references are not treated as ranges or clinical upper limits.
- Shared deterministic Core Text Summary on Dashboard and Analytics, automatically visible at the bottom, editable, with full-text Copy and regenerate confirmation. Drafts are in component memory, not browser storage. Route changes/Back/Forward use styled unsaved confirmation; browser close/reload uses the browser’s native prompt. Session expiry clears draft data; ordinary page unmount clears it.
- Summary owner authorization, at most 366 inclusive days; all paginated nutrition/target/wellness/health reads with explicit bounds. DAY health uses local midnight/calendar-day boundaries, including DST; RANGE health uses UTC bounds. These differ intentionally from nutrition Tracking Days and are printed in the text.
- Every represented nutrient, exact known subtotals/coverage, historical accepted targets/reference points, compaction labels, drinking water, actual weight measurements, activity, scheduled medication/supplement/task facts and status counts. Configured dose is never represented as actual intake. Overdue means no intake/completion recorded. No email/name/DOB, free notes, instructions, AI calls or summary persistence/logging.
- Health requests chunk to 31 days and deduplicate occurrence IDs. At 10,000 records/occurrences/target revisions or foods, affected sections are explicitly incomplete and ask for a narrower range. Unavailable data never silently become zero. Summary generation is a multi-read projection, not a transactional historical export.

## Earlier gated state (superseded for calculated reset below)

- Approval of adult eligibility/exclusions, activity-to-EER mapping/branches, and general-versus-training protein policy/point-reference semantics.
- Consequently personalized profile-based automatic proposals, calculated-default reset and source-range acknowledgements remain disabled. Manual selected values and explicit calorie refresh work. No hidden FDA fallback or automatic water formula.
- Full acceptance-matrix coverage (large 10,000-row truncation, every upstream failure mode and account-switch/browser permission combinations) is not claimed by the initial tests. Add further fixtures as policy-dependent functionality is enabled.

## Verification

- Frontend TypeScript/build; decimal/model tests; API and Core Go tests against isolated disposable Mongo replica-set databases; OpenAPI/protobuf contract validation.
- Added exact 3140-kcal arithmetic, optional-input defaults/conversions, version-conflict/ownership/unchanged-field tests, summary privacy/empty-state/55-record pagination tests.
- Final checks passed: API/Core Go suites, 16 frontend model/regression tests, TypeScript/Vite build, OpenAPI (31 operations/120 references), protobuf descriptors and `git diff --check`. Vite emits a non-blocking ~505-kB JS chunk-size warning; code splitting is an optimization follow-up, not a suppressed build failure.
- `scripts/phase-2.5-local-smoke.mjs` exercises actual loopback API/Core with a synthetic `example.invalid` account. It never contacts production or USDA. It intentionally leaves that fixture account for local browser QA; no actual user records are modified.
- Browser verification notes are appended after local Chrome checks. Policy-dependent features remain disabled irrespective of test results.

### Local Chrome evidence

Verified automatic summary placement on Dashboard and Analytics; full Copy (including offscreen text); edited regeneration cancellation; styled navigation and browser Back confirmation; effective assumed-input labels; explicit macro refresh `160/400/100 → 3140` without saving or changing fibre; explanation-page navigation; desktop light/phone-width dark layout without horizontal overflow. Phone notice wrapping was corrected during QA. Test-only target drafts were restored without saving; theme/viewport overrides were restored.

Long browser checks also exposed access-token refresh reusing the rejected token. Fixed with a regression test that preserves an already-refreshed concurrent token. Expiry/logout bypass unsaved-route blocking and clear memory rather than leaving the app at “Checking session”. Full clipboard-denial, all account-switch cases and the complete acceptance matrix remain unclaimed. Fixture accounts remain only in the local database.

## User-facing guide and badge follow-up — 2026-10-21

Replaced the engineering/policy dump with a structured user guide: exact 4/4/9 formula, live calculation from saved macros, saved/reference-value comparison, worked example, fixed fallback weight derivation, independent-field behaviour, profile/logged-weight distinction, rounding and drinking-water explanation. Internal version/policy/approval metadata and raw enum labels are not rendered. The page explicitly says personal energy needs are not currently estimated from profile data; calculation behaviour itself was not changed.

The Settings badge now has dedicated high-contrast selected/unselected styling, a right-aligned 24px counter, accessible description and tooltip. The missing-details notice includes defaulted inputs so its list agrees with the counter (verified two missing details for sex/activity).

Verification: 18 frontend tests, including two guide-rendering regressions; TypeScript/Vite build and whitespace check pass. Local Chrome verified light/dark badge contrast, live `50/275/79 → 2011 kcal` arithmetic, expandable worked formulas and phone-width wrapping without horizontal overflow. Original theme and viewport restored. No data, calculation formulas, backend, production or remote Git changes made in this follow-up.

## Exact-field Settings guidance follow-up — 2026-10-21

The Settings counter, top-of-page checklist, nutrition notice and inline field markers now share one missing-profile-details model. A single notification links directly to that field; multiple notifications link to the checklist. Checklist/notice controls reveal the containing details section, scroll to the field and move keyboard focus. Missing fields have persistent borders, “Not provided · optional” text, current-fallback explanations and accessible descriptions. Missing optional fields are visible rather than hidden inside a collapsed section.

Indicators describe saved data: selecting a value does not clear them until Save profile succeeds. An explicit “Prefer not to provide” answer completes that optional field; it is no longer counted as missing. Ordinary tracking remains available. No formulas, persisted health records or production behaviour changed.

Verified locally: complete-profile state has no badge; synthetic one-missing-field account deep-links to/focuses Sex; badge remains 1 before saving, and badge/checklist/highlights all disappear after saving DECLINED. Three shared-model tests cover single/multiple/default/explicit-decline cases, alongside the frontend suite and build. Only the disposable local fixture profile was changed in browser QA.

## Activity-driven calculated reset — 2026-10-21

The owner resolved the activity question: use saved activity or Moderate, with no General/Training switch. Core now implements the eight adult National Academies 2023 EER branches and a documented app approximation mapping activity to protein rate (0.8/0.8/1.2/1.6/2.0 g/kg). Moderate and Active use the Active energy branch. This mapping is an implementation approximation, not a source-prescribed clinical conversion. Fat uses 27.5% of energy; carbohydrate uses the remainder; fibre uses 14 g/1,000 kcal. Conflicting carbohydrate shares block reset with a reason, rather than silently clamping.

Missing height/weight/activity use the agreed constants; age and equation-specific sex are never invented. Adults 19+ with supported inputs can preview all five current/new values. Applicability acknowledgement excludes pregnancy/breastfeeding and prescribed diets/fluid restrictions. Cancel preserves saved targets and unsaved drafts. Confirm performs an owner/profile/input fingerprint check plus optimistic active-version guards inside the save transaction, creating a new accepted revision. Profile changes do not silently reset; wellness weight, water and logged food remain untouched. Canonical calculations use rational arithmetic with 16 fractional digits for persistence; preview alone rounds to two decimal places.

The user-facing guide now shows the selected equation and current inputs, activity mapping, macro/fibre formulas and sources. It does not render internal calculation version IDs. Source-range custom-value acknowledgements and automatically created initial proposals are not part of this reset change.

Verification for this follow-up: Core/API Go suites pass (Core uses isolated Mongo replica-set fixtures), 22 frontend model/rendering regressions pass, TypeScript/Vite build passes, and OpenAPI/protobuf validation passes (32 operations / 123 references). The expanded local smoke script confirms preview is read-only, all five reset fields match the server calculation, wellness weight never becomes profile weight, and replaying the stale request returns 409. Chrome extension testing on a disposable local account verified a saved protein override of 90 g versus the 84.53 g displayed calculation, draft-preserving Cancel, applicability-gated confirmation and successful revision save. Desktop and 390-pixel mobile layout checked; dialog now initially focuses its heading at scroll position zero instead of jumping to the checkbox. Viewport restored. Existing non-blocking Vite chunk-size warning remains. No remote push or production deployment.

## Text Summary presentation follow-up — 2026-10-22

Dashboard and Analysis share a redesigned summary card: collapsed by default with an accessible disclosure, full-copy and regenerate actions outside the hidden content, a sticky action header while expanded, readable date/context/status metadata, consistent padding and an explicitly theme-styled editor. The full report remains editable and copyable; collapsing never discards its draft. Regeneration still asks before replacing edits. Clipboard denial expands/selects the editor rather than leaving selection hidden. Existing report text remains copyable during refresh.

Verified on a disposable local account using the Chrome extension: closed-by-default Dashboard and Analysis cards, dark/light editor contrast and padding, full edited clipboard contents while collapsed, cancelled regeneration preserving edits, successful regeneration and 390-pixel mobile layout without horizontal overflow. Theme and viewport restored. Four new presentation regressions join the 26-test frontend suite; TypeScript/Vite build and whitespace checks pass. No backend/data changes, push or production deployment in this presentation follow-up.

## Analysis calendar follow-up — 2026-10-22

Replaced the Analysis browser-native date inputs with the shared app calendar, extracted from Settings/History into a reusable component without importing App into a feature. Dates display as 15 Oct 2026; month/year controls, aligned weekday/day grids, theme styling, current/selected states, bounded range days, Today/Close, outside-click and Escape dismissal are provided. The popup renders outside animated/overflow containers and clamps to the viewport; mobile positioning is centred. Only one shared picker opens at a time.

Local Chrome verified desktop dark and phone-width light rendering, year selection, date selection and disabling dates before the selected start. Theme and viewport restored; only disposable-account filters changed. Two rendering regressions cover accessible controls, canonical ISO form submission values, leap-day formatting and no native date input in the shared widget. TypeScript/Vite build passes. This fixes the reported Analysis calendar; it does not claim all remaining native date fields elsewhere have been migrated.

## Phase 3 and later

No change to the separately approved Phase 3 scope. No Redis, new server, paid AI service, barcode scanning or data-recovery guarantee added here.
