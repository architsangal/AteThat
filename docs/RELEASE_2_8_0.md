# v2.8.0 — Food Library and focused nutrition

## Release scope

- Food Library holds reusable individual foods; Recipes combines foods and includes older saved meals. My foods, Quick log and Search share a mixed meal selection with fractional recipe portions.
- Quick log accepts nutrition for the amount eaten without requiring a reusable food. Saving to Food Library is optional. Empty visible fields become zero only when that explicit option is selected; hidden nutrients remain unknown.
- Settings selects tracked nutrients across logging, Dashboard, Analysis, recipe reviews and Text Summary. Energy remains included; full source nutrition is preserved.
- Analysis loads the latest seven account Tracking Days automatically. Explore applies later range changes to nutrition and health charts.
- Account email/link status, clearer public wording, accessible theme-aware checkboxes, larger click targets and a labelled recipe-serving dialog improve everyday use.

## Verification

- Production web build and 56 frontend regressions pass.
- Full API and Core suites pass, including disposable local replica-set integration tests. macOS tests use `-ldflags=-linkmode=external` to work around the local internal-linker loader issue.
- OpenAPI and Protocol Buffer checks pass.
- Chrome review covered manual logging/reuse, mixed fractional recipes, tracked selections and hidden-nutrition preservation; desktop/390px mobile, light/dark themes, keyboard controls and weekly/monthly schedule choices were checked locally. Synthetic local provider data was used; no production health data was changed for these tests.
- Non-blocking existing web bundle-size warning remains; emailed/PDF reports remain Phase 3.

## Deployment and compatibility

Promote through feature PR → dev → main with passing required checks. Backend deploys through the existing restricted GitHub Actions production workflow; Cloudflare independently deploys the frontend. Published GitHub release notes record the promoted commit and observed deployment status.

No destructive database migration or new collection is required. Existing saved meals without a yield behave as one serving. MANUAL food snapshots and the profile email response extend existing contracts; deploy API/Core and frontend together. Keep full nutrition and immutable historical snapshots.

Previous code checkpoint: v2.7.1 (`104fce4`). Reverting code does not erase data or constitute a database restore; manual logs created by v2.8.0 remain stored and require compatible readers. Prefer a reviewed forward fix over an unexamined rollback.

See [detailed scope and verification](FOOD_LIBRARY_AND_TRACKED_NUTRIENTS.md).
