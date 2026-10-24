# Food Library, food logging and tracked nutrients

Release scope: v2.8.0, 24 October 2026. See [release verification and compatibility](RELEASE_2_8_0.md); published GitHub release notes establish promotion and production deployment status.

## Phase 1 and Phase 2 — compatibility

Existing custom foods are individual Food Library entries. Existing saved meals appear with recipes; a missing historical recipe yield means one complete serving. Fractional servings are supported without rewriting old definitions or food logs. Existing `/foods/custom`, `/meals` and `/app/meals` paths remain compatible.

## Current implementation

- Main navigation uses **Food Library** for individual reusable foods and **Recipes** for combinations.
- Log food opens **My foods**, with Food Library and Recipes pickers. **Quick log** and **Search** share the same editable meal selection. Multiple recipe portions append ingredients instead of replacing the selection.
- Quick Log accepts a name, amount in g/ml, nutrition for that amount, meal category and optional log time. Blank visible nutrient fields count as zero when the explicitly labelled checkbox is checked; unchecked blanks and hidden fields remain unknown. Additional one-off nutrients can be added without changing Settings.
- One-time foods use a `MANUAL` input with a `manual_food` payload, validated in Core with the same name/quantity/state/nutrient rules as custom foods. Preview/confirmation, immutable snapshots, idempotency, ownership and corrections follow the existing pipeline. A one-time entry does not create a custom-food document and is excluded from reusable recent/frequent references.
- Optional **Save to Food Library** creates a private reusable food. Search offers a non-blocking reuse prompt. Saving a combination as a recipe preserves manual and edited nutrition by materializing private ingredient definitions before saving references. New combinations default to one serving and can be edited on Recipes.
- Shared catalogue search is called **AteThat foods**. USDA attribution and the existing discovery allowance remain visible.
- **Tracked nutrients** in Settings persists the existing `dashboard_nutrients` preference as an app-wide selection. ENERGY is required; the default is energy, protein, carbohydrates, fat and fibre. Select all supports all 49 recognized nutrients, including B6/B12, with backend validation against duplicate or unsupported codes. Independent preference edits preserve other saved preferences and retain version checks.
- Dashboard, Analysis, input forms, food review, recipe summaries/comparisons and Text Summary use the selection. Full source nutrition stays stored; expandable All nutrition details remains available. Editing a Food Library entry preserves nutrient fields that were not displayed/submitted.
- Settings displays the authenticated account email and Google-link state. The email is a read-only profile response field obtained by Core for the authenticated user.
- Terms use plain recovery-limit wording without storage provider/tier details or obsolete local-testing text. Food search failures and the setup guide use product terminology. Text Summary omits signed occurrence identifiers and opaque target-policy/reference version codes.
- Analysis automatically loads the inclusive last seven account Tracking Days; Explore applies later range changes to nutrition and health charts.

## Verification

Production web build, frontend regressions, API/Core tests, local replica-set integration tests and OpenAPI/protobuf checks are required. Local browser checks cover one-time logging, saving and reusing foods, mixed fractional recipe portions, persistent tracked selections and preserving hidden nutrition on edit. Use disposable local fixtures only.

Verified locally: production web build and 56 frontend regressions; API and full Core suites (Core includes local replica-set integrations); OpenAPI/protobuf validation. Chrome exercised one-time confirmation without a library entry, optional food saving, a mixed recipe saved from reusable/manual foods, combined 0.5 and 0.25 recipe portions totalling 142.5 kcal, all-49 selection saving and a persisted energy/protein/vitamin-C selection. An imported fixture edited from 90 to 91 kcal retained its hidden 23 g carbohydrate, 0.3 g fat and 2.5 g fibre. Search/import used a synthetic provider on localhost; no live USDA request or production record was involved.

No automatic email delivery or PDF generation is part of this implementation.

### Checkbox and wording review

All app checkboxes share theme-aware rounded controls, visible keyboard focus and consistent sizing. Nutrient choices, Quick Log options and health-chart choices use larger clickable labelled rows; weekly and monthly schedule choices retain their specialized layouts. Disabled Energy is labelled Always on. Selected counts and unsaved changes are visible without changing saved preferences.

Quick Log explains zero versus unknown values, uses Review meal and keeps confirmation explicit. The nutrient toolbar uses Energy only and Use defaults. Recipe portions use a labelled dialog with fractional servings and inline validation, rather than a browser prompt. Settings describes account, nutrition and appearance preferences in plain language.

Chrome review covered desktop and 390px mobile layouts, light/dark themes, keyboard toggles, weekly/monthly choices, health-chart selection and adding a half-serving recipe draft. No meal was confirmed or production record changed; existing saved selections and the System theme preference were preserved. The production web build and all 56 frontend regressions pass.

## Phase 3 — deferred report delivery

The explicitly requested emailed PDF Analysis report is in [the Phase 3 backlog](PHASE_3_BACKLOG.md). Normal page loading does not send email. Delivery controls, report limits and duplicate/failure handling remain design details for that phase.
