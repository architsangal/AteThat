# AteThat — calculated targets and factual Text Summary

Prepared: 2026-10-20. Audience: the software-engineer chat.

## 1. Authority, status and working instructions

### Owner clarifications — 2026-10-21 (override conflicting earlier proposals below)

- The calculation explanation is a **user guide**, not an engineering status page. Show the formulas actually used, worked arithmetic, the user’s saved values, sources and assumptions. Do not display internal policy identifiers, version numbers, pending-approval lists or raw enum labels. Internal metadata stays in API/storage/docs. Do not imply a personalized formula is running when it is not.
- The Settings missing-details badge must be readable in both themes and agree with the details listed in the notice, including details using a default.

- Missing height: constant **175 cm**. Missing profile weight: constant **70.4375 kg**, selected once from BMI 23 at 1.75 m; never compute BMI to supply this fallback at runtime. Missing activity: **Moderate**. Supplied profile values take precedence. Show assumed versus supplied inputs. Do not persist a fallback as a measured weight.
- Wellness weight logs never update profile calculation weight. Saving profile changes prompts a target review; accepted targets remain unchanged until explicit replacement.
- Latest owner decision: activity directly selects the rate; use Moderate when absent. Do not add a separate General/Training toggle. Implement previewed, explicit calculated-default reset using supplied data and the approved fallback constants. No default age or sex: if absent/declined, explain which formula input is unavailable and keep manual targets usable.
- The implementation makes the activity mapping explicit as an **AteThat approximation**, not a scientifically measured PAL or a clinical prescription: Sedentary → Inactive / 0.8 g/kg; Light → Low active / 0.8 g/kg; Moderate → Active / 1.2 g/kg; Active → Active / 1.6 g/kg; Very active → Very active / 2.0 g/kg. Moderate and Active deliberately share the adult EER branch. These concrete mapping choices are implementation assumptions; sources do not prescribe those exact UI bins.
- Adult EER uses all eight Table 5-16 branches; age must be 19+. Reset requires applicability confirmation (not pregnant/breastfeeding, no prescribed diet/fluid restriction). No medical history is collected. Fat = 27.5% energy / 9, carbohydrate = remaining energy / 4 (reject conflicting 45–65% share), fibre = 14 g/1,000 kcal. Water is excluded. Calculations use rational arithmetic with up to 16 fractional decimal digits for persistence, not UI-rounded values.
- Preview is read-only; confirmation sends a server-recomputed owner/profile/input/calculation fingerprint plus active ID/revision and profile version. A transaction replaces all five targets as CALCULATED_DEFAULT, snapshots assumptions, and preserves history; stale/failing requests have no partial writes. No automatic reset occurs after profile saves.
- The earlier **persistent** mismatch notice in section 2.6 is superseded: show the non-blocking notice after macro edits, allow dismissal, and clear it on page reload. Keep independent field saves and deliberate 4/4/9 calorie refresh.
- Implementation/evidence and remaining gates are maintained in [Phase 2.5 status](PHASE_2_5_STATUS.md). Local work only; no push or deployment authorized by this document.

This handoff captures two features discussed with the owner. The interaction requirements below are agreed. The Phase 2.5 label is a proposed grouping, not a published release. Formula/eligibility choices explicitly marked as pending need owner approval and source/clinical validation before enabling personalized recommendations; the engineer must not invent those policies.

- Feature 1: profile-based initial nutrition targets, independent custom targets, explicit calorie refresh/reset, persistent warnings and an explanation page.
- Feature 2: editable, factual Text Summary displayed by default at the bottom of Dashboard and Analytics.
- Assistant/MCP integration is a separate future feature. This handoff does not authorize building it or sending data to an AI provider.
- Read the current code and maintained docs before changing contracts. The workspace already contains another chat's uncommitted Phase 2.1 work, including profile, food and health-task changes. Preserve it; do not reset, switch branches over it, overwrite it or commit unrelated edits. Coordinate the working branch before implementation.
- Apply the existing workflow: feature branch based on `dev`, local verification first; PR/push/merge/deployment only on explicit owner authorization. No production writes for routine testing.
- Keep API → private Core → MongoDB ownership, authenticated isolation, decimal strings/Decimal128, version guards and immutable historical target comparisons. Extend formal contracts and generated consumers together where necessary.

## 2. Feature 1 — owner-agreed behaviour

Latest presentation override: Text Summary is collapsed by default on Dashboard and Analysis, with Copy full summary and Regenerate summary always accessible. Expanding reveals the editable report; collapsing preserves its draft. This supersedes earlier always-expanded presentation wording.

### 2.1 Names and placement

- Settings is the main editing location. Add a coherent nutrition-target section rather than separate competing editors with different rules.
- The editable energy field is **Daily calorie target**. Do not label every custom target as weight maintenance, weight loss or weight gain.
- The automatically calculated starting calorie value uses a maintenance estimate. Users can independently select higher/lower targets; no automatic deficit/surplus or goal-based adjustment is agreed.
- For each supported nutrient show its selected target, unit, applicable reference range/value and provenance: calculated default or custom. Describe a reference range as guidance, not a clinical safety limit.
- Dashboard cards use the accepted selected targets, not a proposal, range midpoint recalculated in the browser or an obsolete default.

### 2.2 Profile prerequisites and automatic proposal

- Reuse stored profile values wherever possible: DOB/age, height and weight. Convert units in Core; do not ask twice for an existing usable value.
- Additional necessary information includes the sex category required by the equation and a clearly defined physical-activity category. Protein training criteria are a separate input/policy; do not infer them from an undefined “moderate” activity label.
- Missing required details produce a small, accessible Settings/navigation badge and an inline list linking to the missing fields. Do not rely solely on colour or an icon without a label.
- Profile completion is optional for ordinary tracking. Missing/declined inputs must not block food logging or erase existing targets. Do not substitute invented age, sex, height, weight or activity.
- Once required eligible inputs exist, calculate a proposal automatically without asking users to calculate their own numbers. Show the values/basis and use the existing explicit target-acceptance workflow; profile saving must not silently replace accepted targets.
- A profile change marks calculated recommendations as needing review. Existing accepted/custom values remain active until the user explicitly saves a replacement or resets.
- Source-of-weight policy is resolved above: use saved profile weight or the fixed fallback, never a wellness log. Display the chosen weight and its date/source in the proposal.

### 2.3 Defaults versus editable targets

- Where an approved reference is a range, propose a documented central value where appropriate. Example discussed: a protein range of 1.2–2.0 g/kg has midpoint 1.6 g/kg, giving 160 g at 100 kg when that training policy applies.
- Do not independently choose every macro midpoint if that produces an incoherent energy total. The proposed initial algorithm in section 3 chooses protein first, fat second and carbohydrates from the remaining energy.
- After initial acceptance, each manual edit affects only the edited field. Changing protein must not rebalance carbohydrate, fat, fibre, water or calories. Changing calories must not automatically change other targets.
- Keep the untouched decimal values; do not replace all fields with rounded browser-display strings on save.
- Allow a user to retain/enter a valid custom target outside the reference range after explicit confirmation. Do not present that confirmation as medical approval.

### 2.4 Range warnings and acknowledgements

- Editing/saving a value outside its applicable reference range requires a styled confirmation identifying the nutrient, selected value, unit and range: “This target is outside the reference range shown. Save this custom target?” Cancel preserves the existing accepted values.
- Acknowledgement applies only to the exact edited values and reference/proposal revision. Changing either invalidates that acknowledgement; backend rules must not rely on a browser-only flag.
- A confirmed out-of-range target remains visibly labelled custom/outside reference when Settings reopens. Do not require another modal merely for viewing an unchanged, previously acknowledged value.
- Validate amounts/types/units on the server. Confirmation never bypasses invalid input. Proposed numeric contract: calorie targets must be positive; macro/fibre amounts finite and nonnegative, with zero handled as outside guidance where applicable. Preserve the existing optional-water-goal clearing behaviour. Technical bounds must be documented separately from medical warnings.
- A point reference (for example an RDA) is not a two-sided range or a tolerable upper limit. Do not fabricate an upper boundary to trigger this interaction. Final lower/reference warning semantics for such points are a policy gate.
- Reference comparisons for a saved proposal must state their energy/input basis. For custom calories, showing a range at that calorie level is a comparison-only calculation; it must never mutate other selected targets. If confirmation changes the range basis, evaluate acknowledgements again against the resulting saved state.

### 2.5 Explicit calorie refresh

- Place an accessible refresh/recalculate control beside Daily calorie target, with explanatory text: **Calculate calories from selected macros**.
- Only a deliberate click updates the calorie input. Formula:

  `macro_calories = 4 × protein_g + 4 × carbohydrate_g + 9 × fat_g`

- Use the current valid form values, including unsaved edits. If a macro input is absent/invalid, disable refresh and explain which value is required; do not interpret blanks as zero.
- Refresh changes only the calorie input. It does not save immediately or change any macro, fibre, water, profile detail or log. The user saves through the normal form.
- Mark the accepted calorie value as custom/macro-derived rather than profile-calculated. Retain the profile-based maintenance estimate separately as explanatory context, not as a compulsory target.
- The 4/4/9 calculation is a target-planning convention, not a replacement for recorded food energy. Do not rewrite consumed calories from nutrient totals: food labels/provider data can differ because of fibre, alcohol and rounding.

### 2.6 Persistent mismatch notice

- If selected calorie target and selected macros' 4/4/9 energy disagree beyond the published rounding tolerance, display a non-blocking notice containing both values and the difference.
- The notice is derived from current saved/form values, not a one-time toast. It reappears when Settings is reopened/reloaded while the mismatch still exists, even if the user saved intentionally.
- Users may save valid mismatched custom targets; this notice does not force a rebalance or repeatedly block them.
- Explicit refresh resolves this mismatch if saved consistently, but does not dismiss other applicable range warnings.
- Proposed technical tolerance: compare canonical values, accepting differences up to 1 kcal from documented rounding. Owner can change this before implementation; use one tested server/client rule.

### 2.7 Reset

- Add **Reset to calculated defaults** with a preview/confirmation of the fields that will change.
- Reset recomputes the approved defaults from the current eligible profile and formula version, not the account's original profile or a hidden FDA fallback.
- Only confirmation plus successful persistence replaces accepted custom targets. Cancel/failure leaves them intact.
- Missing/unsupported inputs link to profile completion and prevent reset; ordinary logging remains available.
- Reset does not alter logs, historic target revisions or profile inputs. Water must remain unchanged until its automatic-goal policy is approved; list that exclusion in the confirmation.

### 2.8 Calculation explanation page

Provide a dedicated **How your targets are calculated** page, linked from both Settings and Dashboard target sections. It must show:

1. The actual inputs used, canonical units and calculation date.
2. Formula name/version, applicability and source links.
3. Selected equation, substituted values and calculated result.
4. Reference ranges, the method used to select defaults and which fields were customized.
5. Why default calories use a maintenance estimate while custom targets may serve other goals.
6. Explicit calorie-refresh formula and the fact that manual edits do not rebalance anything.
7. Rounding, uncertainty, unsupported cases and water-source distinctions.
8. A clearly labelled fictional 100 kg worked example, separate from the signed-in user's own calculations.

The explanation and default calculations must share the same versioned Core policy. Do not hard-code conflicting formulas in explanatory UI copy.

## 3. Original formula proposal and approval gates (historical context)

The owner clarifications above supersede conflicting training-toggle and disabled-reset proposals in this historical section. Broader clinical applicability and source-range acknowledgements are not claimed implemented. These are researched candidates discussed in this chat, not permission to ship a clinically personalized engine without reviewing applicability/source reuse. Current released FDA general benchmarks are not already personalized targets. Never relabel or silently migrate them as if they were.

### 3.1 Proposed initial scope and inputs — approval required

- Generally healthy adults aged 19+, not pregnant/breastfeeding and without prescribed dietary/fluid restrictions. Confirm the exclusions and consent wording with the owner; do not collect a full medical history unnecessarily.
- DOB-derived completed age, selected current weight in kg, height in cm, sex category required by the reference equation and one of its defined activity categories.
- Explain the sex input as formula-specific, not an inferred gender identity. If the user declines or no suitable equation applies, offer manual targets and do not guess.
- Decide source-backed activity definitions and how the user selects them before enabling calculation. Never infer energy category solely from a gym-session count.
- Confirm protein mode/eligibility, selected-weight source and point-reference warning policy. Broad energy activity and athletic protein requirements are not interchangeable.

### 3.2 Calories

Candidate source: National Academies, **Dietary Reference Intakes for Energy (2023), Table 5-16**, adult equations. Use the table's coefficients for the selected sex/activity category; no extra BMR activity multiplier on top of EER. Source: https://www.nationalacademies.org/read/26818/chapter/7

For the fictional male/active example (age 30 years, 180 cm, 100 kg), that equation yields approximately **3,445 kcal/day**. Confirm all supported equation branches against source fixtures before enabling them. Do not use the male/active equation for other categories or derive a weight-loss surplus/deficit automatically.

### 3.3 Macronutrients and fibre

| Quantity | Candidate reference/calculation | Selection behaviour |
| --- | --- | --- |
| Protein, general adult reference | 0.8 g/kg/day RDA reference | A point reference, not a range midpoint, athletic optimum or upper safe limit. General-mode default/warnings require approval. |
| Protein, qualifying regular training | 1.2–2.0 g/kg/day; midpoint 1.6 g/kg/day | Candidate default for an explicitly approved training mode, not everyone who selects “moderate”. |
| Fat | 20–35% of reference energy; grams = energy × fraction / 9 | Proposed initial default uses 27.5%. |
| Carbohydrate | 45–65% of reference energy; grams = energy × fraction / 4 | Proposed initial default is remaining energy after selected default protein and fat, not independently 55%. |
| Fibre | 14 g per 1,000 kcal reference density | Candidate calorie-scaled estimate, not a fabricated clinical min/max interval. |

Do not combine all range upper endpoints as one plan. Validate that an initial default plan fits the chosen reference guidance and positive remaining energy. If it cannot, return an unsupported/conflicting proposal with reasons; do not silently clamp a nutrient or invent an acceptable range. The engineer must resolve the general-versus-training protein policy with the owner before shipping this branch.

Keep any existing vitamin/mineral references attributed and separate. This feature does not authorize new body-weight multipliers for all micronutrients, sugar or sodium.

Sources for verification:

- Adult macronutrient reference tables: https://www.ncbi.nlm.nih.gov/books/NBK208874/
- General adult protein reference context: https://www.nationalacademies.org/read/9620/chapter/2
- Training protein guidance: https://www.medlineplus.gov/ency/article/002458.htm (follow its cited athletic-performance reference when validating the policy).
- Fibre-density reference discussion: https://www.nationalacademies.org/read/11767/chapter/9

### 3.4 Water — unresolved, do not invent a multiplier

Total-water reference examples of 3.7 L/day for men and 2.7 L/day for women include food and other beverages. They must not become automatic plain-drinking-water goals in the existing tracker. A weight-based multiplier was not agreed. Preserve the existing user-selected optional drinking-water goal until the owner approves a source-backed policy and what the tracker counts. Explain this boundary on the calculation page; do not present water as automatically personalized yet.

Source: https://www.nationalacademies.org/read/10925/chapter/2

### 3.5 Fictional 100 kg example for explanation/testing

Illustrative inputs: 30-year-old male, 180 cm, 100 kg, active energy category and qualifying training protein mode. This is a worked example, not a recommendation for every 100 kg person.

Using the approximate displayed energy estimate 3,445 kcal for simple arithmetic:

- Protein reference range: 120–200 g; midpoint/default 160 g = 640 kcal.
- Fat default: 27.5% × 3,445 / 9 ≈ 105.3 g.
- Carbohydrate default: (3,445 − 640 − 947.375) / 4 ≈ 464.4 g, within the corresponding 45–65% reference-energy interval.
- Fibre-density estimate: 14 × 3,445 / 1,000 ≈ 48.2 g.
- If the user chooses protein 220 g, request outside-reference confirmation. On saving, carbohydrate, fat, fibre and calories remain untouched; the saved settings show the resulting mismatch.
- For an exact refresh fixture independent of the profile: protein 160 g, carbohydrate 400 g, fat 100 g produce **3,140 kcal**. Clicking refresh changes only calories to 3,140; saving preserves all three macro inputs and the fibre/water values.

Production computation uses canonical unrounded decimals with a published display-rounding rule, not successive rounded example values. Preserve this distinction in snapshots/tests.

## 4. Feature 1 — persistence and acceptance

- Keep the existing proposal/accept/reject/effective-date history model. Version-check writes; a stale settings tab cannot replace newer targets or acknowledgements.
- Record input snapshot/source/date, reference/formula version, recommended reference values/ranges, selected values, per-field provenance and applicable warning acknowledgements. These are required semantics; reuse current entities where possible instead of creating redundant stores.
- Save a target set atomically. If water remains a separate versioned settings entity, show separate save outcomes honestly; do not fake a cross-service atomic save. No water reset is needed in this scope.
- Historical Dashboard/Analytics must select the accepted target effective for each Tracking Day. Current references must not rewrite past comparisons or change unknown values to zero.
- Locally test unit conversion, activity/sex formula fixtures, missing/declined/unsupported inputs, range boundaries, zero/invalid values, decimal rounding, stale versions, confirmation cancellation, acknowledgement invalidation, persistence after reload, exact refresh, independent manual edits and reset cancellation/failure.
- Verify repeated profile updates never overwrite custom values, and edits by one account cannot affect another.
- Formula-specific release gate: owner approves the remaining policy items and numerical/source review before enabling personalized defaults. Settings mechanics and Feature 2 can be developed/tested independently; label unfinished calculation branches honestly.

## 5. Feature 2 — Text Summary (owner-agreed)

### 5.1 Placement and controls

- Heading: **Text Summary**. Not “AI prompt”, “Ask AI” or “Recommendations”.
- Display by default at the end of the Dashboard and at the end of Analytics, after their normal content. The text must be visible in an accessible editable area, not hidden in a collapsed section or behind Generate.
- Dashboard summary describes the displayed Tracking Day, not necessarily the device's current calendar date.
- Analytics summary describes exactly the selected date range and applicable filters; state which filters affect which sections rather than pretending one nutrient filter limits medication data.
- Plain editable text with **Copy** and **Regenerate from data** controls. Manual edits affect only the text draft, never saved records or targets.
- Copy the complete current edited text, including content outside the visible text-area viewport, without Markdown fences, wrapper instructions or added commentary. Report clipboard failure honestly and permit manual selection/copy.
- Regeneration replaces the text with the latest factual data only after confirmation if edits would be lost. Cancellation retains edits. No confirmation is needed when the text is unchanged.
- Initial generation is automatic once relevant data is loaded; no extra button click is required. If filters/day/data change while the draft is dirty, retain it, mark its old date/filter context and offer regeneration; never silently overwrite or relabel an old draft as current.
- Keep these health-data drafts in memory for the current screen/session by default; no new server persistence, localStorage, analytics payload or external transmission is required. Protect/discard an edited draft on navigation with the existing styled unsaved-changes interaction. Clear it on logout/account change.

### 5.2 Required factual content

Include the available facts below, with explicit units and date/timezone context. Use stable ordering; never generate narrative interpretations with an LLM.

| Section | Required facts |
| --- | --- |
| Scope | Tracking Day or selected range, account timezone/daily reset for nutrition, generation time and relevant filters. |
| Nutrition | Consumed energy, protein, carbohydrate, fat, fibre and every other nutrient represented in the source analysis data; corresponding accepted selected targets where available. Additional nutrients may use grouped plain-text lines, but no silent top-five truncation. |
| Target context | Selected/custom target, applicable reference guidance where available and whether targets changed within the selected period. Do not invent one target for an entire range with several accepted revisions. |
| Water | Logged drinking-water amount and selected drinking-water goal if present; do not label it total dietary water. |
| Weight | Actual measurements with dates; for a period, count/first/latest and correctly scoped change if meaningful. No carry-forward “today's weight” unless explicitly labelled last recorded. |
| Activity | Recorded activity names, durations, intensity and optional logged estimated energy. Missing estimated energy is unknown, not zero. Do not subtract it from nutrition totals automatically. |
| Medication and supplements | Recorded item names/types, configured dose/unit when present, scheduled date/time/timezone and recorded occurrence status; period summaries include due/status counts and identifiable item-level facts. |

Medication/supplement rules:

- A configured/scheduled dose is not evidence of an actual amount ingested. Label it scheduled/configured unless an actual dose was explicitly recorded.
- Preserve taken, skipped, rescheduled, upcoming and unrecorded status distinctions supported by the backend. If the app derives an overdue/missed state from no action, label it **overdue — no intake recorded**, not a confirmed missed dose or a medical judgment.
- Taken status may be reported as recorded taken; never infer clinical adherence, effectiveness or dosage advice. Show action time only when recorded, and original/new occurrence links when rescheduled.
- Do not add free-form health notes, medication instructions, account name/email/DOB or hidden profile fields to the copy by default. Medication/supplement names and dose facts are explicitly included by the owner; show a small note that the summary contains sensitive health information.
- Other current health-item kinds, such as generic tasks, must not be misclassified as medications. They may appear as a separate factual task section if already part of the source panel; they are not required to expand this feature.

### 5.3 Date scope, completeness and unavailable data

- Do not copy the Dashboard's next-24-hour or seven-day health preview and claim it is the displayed day's medication history. Fetch the medication/supplement occurrences for the intended day/range using explicit timezone boundaries. Nutrition uses the saved Tracking Day convention; health schedules use actual instants and schedule timezones. If Analytics source sections use different boundaries (for example UTC health history), label each explicitly rather than silently merging them.
- Fetch all pages within supported bounded ranges. Paginated first-page totals must not masquerade as complete daily/period totals. Aggregate server-side or use the existing bounded complete-fetch pattern where suitable.
- No record is not a known zero. A logged nutrient value of zero remains 0; absent/unsupported values are “Not recorded” or “Unknown” as appropriate. Mixed known/unknown nutrition must expose known subtotals/incomplete coverage rather than claim complete totals.
- Period summaries show recorded/known-day counts and average denominators per metric where relevant. Do not divide every metric by total calendar days if the app's metric uses known recorded days.
- Distinguish “No medication scheduled”, “No action recorded”, “Data unavailable” and “Still loading”. An upstream/API failure must not become an empty successful summary.
- If one section fails, retain other available facts, label the incomplete section and offer retry. Copy remains available only with explicit partial/loading labels; never imply all data was fetched.
- Compacted historic detail must be identified. Available daily totals may be summarized, but deleted foods/doses/events must never be reconstructed or invented.
- Respect existing range limits. No unbounded export is required; for lengthy fact lists, the editable area may scroll but Copy includes all text. Never silently truncate; show explicit limits and request a narrower period if necessary.

### 5.4 Example format — facts only

```text
Text Summary
Tracking Day: 2026-10-20
Nutrition timezone: Asia/Kolkata; daily reset: 04:00
Calories: 2,100 kcal; selected target: 2,400 kcal
Protein: 110 g; selected target: 140 g
Carbohydrates: 260 g; selected target: 280 g
Fat: 69 g; selected target: 80 g
Fibre: Unknown
Logged drinking water: 1,800 ml; selected goal: 2,000 ml
Weight: Not recorded for this Tracking Day
Activity: Walking, 30 minutes; estimated energy: Not recorded
Medication — Example medication A: scheduled dose 1 tablet, 08:00 Asia/Kolkata; status: recorded taken
Supplement — Example supplement B: scheduled dose 1 capsule, 09:00 Asia/Kolkata; status: overdue — no intake recorded
```

All values/names above are fictional formatting fixtures, not prescriptions, defaults or a claim of an internally balanced macro/calorie target. Do not append “review this”, “highlight patterns”, “ask about goals”, advice, questions for an AI or other prompt instructions.

### 5.5 Architecture and acceptance tests

- Use existing authorized data/aggregation services as the factual source. Share one deterministic formatter/scope model between Dashboard and Analytics; no separate AI service, paid provider, new database or MCP dependency.
- Any added summary endpoint must use normal owner authentication, bounded ranges, projections/minimal data and standard errors. Do not let clients submit trusted owner IDs. Choose whether existing APIs suffice after inspecting current code; do not invent a parallel domain calculator.
- Keep amount arithmetic, units and aggregates consistent with Core. Formatters format facts; they do not recompute authoritative nutrition using binary floats or reinterpret medications.
- Unit-test the plain-text formatter with exact fixtures; ensure it has no prompt/recommendation text and preserves known zero/unknown/partial coverage.
- Test medication actions/timezones, Tracking Day crossings, DST where applicable, different Analytics filters, changed historical target versions, all-page fetching, compacted detail, service failures and all-empty data.
- Browser-test default visibility, keyboard access, phone-width wrapping, editable text, Copy including offscreen content, clipboard denial, regenerate cancellation, dirty draft on date/filter/navigation change and account switching. Test both themes.
- Confirm the copy operation causes no network request to an AI service and no sensitive text reaches logs/analytics/browser-persistent storage.

## 6. Delivery checklist for the engineer

1. Inspect current Phase 2.1 work and coordinate branch ownership; do not overwrite it.
2. Identify existing target/profile/wellness/history/health APIs and contract gaps. Report the minimal changes needed before broad refactoring.
3. Surface the formula-policy gates in section 3 for owner approval. No automatic water goal, training criteria, calorie deficit/surplus or clinical target range may be invented.
4. Implement/test the agreed Settings controls, warnings and explanation page with versioned fixtures; keep unapproved personalized policies disabled/clearly pending.
5. Implement/test factual Text Summary on both pages without an AI integration dependency.
6. Update affected API/database/product docs alongside implemented changes and distinguish implemented from pending behaviour.
7. Run relevant frontend typecheck/build/model/browser tests, API/Core tests with isolated local Mongo where needed, contract checks and local smoke tests. Provide test evidence and any remaining policy blockers.
8. Stop for owner review before pushing, PR promotion or deployment. This handoff is implementation guidance, not authorization for remote mutations or production changes.
