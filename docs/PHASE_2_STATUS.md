# Phase 2 status and remaining decisions

Reviewed: 2026-10-16. Source: the Phase 2 sections of `2. PRD.pdf`, reconciled with the owner's subsequent exclusions and the released implementation.

Status: complete and released as [v2.0.0](https://github.com/architsangal/AteThat/releases/tag/v2.0.0), commit `4e649a2a2ca1ba23395f98f415d71ed6d7a6bd03`. Backend production deployment and designated-test-account Chrome sanity completed on 2026-10-16. Barcode scanning is removed from scope, not an outstanding Phase 2 deliverable. The local-first USDA catalogue and quota proposal belongs to Phase 3; expanded Indian-food provider coverage is not a Phase 2 release blocker.

## Phase 1 — completed baseline

Release `v1.0.0` supplied the core tracking/authentication/production foundation; Phase 2 extends it without replacing its ownership or confirmation rules.

## Phase 2 — implemented and deployed

| Area | Available behavior |
| --- | --- |
| Full nutrition analysis | Vitamins, minerals, fats, amino-acid/fibre subtype fields where provider metadata supports them; collapsible groups; per-nutrient trends; known-day averages; per-day accepted target comparisons; food/meal contribution drill-down. Missing data stays unknown. |
| Recipes | Ingredient references, quantities and food states; preparation notes; serving yield; batch/per-serving calculation; fractional portions; edit/delete; per-ingredient nutrient overrides; explicit preview and idempotent confirmation. Historical snapshots remain stable. |
| Saved meals | Reuse as an editable group; adjust quantity and expanded nutrient overrides before a fresh review and confirmation. Existing personal food-specific serving mappings remain available and traceable. |
| Water | Timestamped ml/L entries, edit/remove, Tracking Day history/trend, optional owner-selected daily goal, dashboard consumed/goal progress. No medical water requirement is prescribed. |
| Weight | Timestamped kg/lb measurements, notes, edit/remove, daily trends and first/latest change over a selected range. |
| Activity | Named activity, duration, intensity, optional estimated kcal, notes and history/trends. Estimates do not automatically alter nutrition targets. |
| Scheduling | Daily/weekly/monthly intervals, multiple times, selected weekdays/month dates, end dates, versioned pause/resume, schedule history including inactive items, adherence counts and taken/due percentage. Changes cannot rewrite past or today's schedule dates. |
| Data safety | Owner isolation, optimistic version guards, Decimal128 amounts, saved Tracking Day context, wellness deletion with the account, immutable confirmed food snapshots. |
| Supporting UX | Date-of-birth month/year selection and grouped nutrient-entry controls. Terms describe the current lack of a recovery guarantee. |

The water goal is user-selected, not an automatically personalized clinical reference. Schedule pause/resume applies to whole future dates; an ended pause does not resume automatically. Health-history date filters explicitly use UTC, while rows show saved schedule-local dates and times.

### Scope decisions following owner review

- Barcode scanning and product lookup are removed from planned scope. Do not build them or treat barcode-provider licensing as a Phase 2 blocker.
- Local-first food search, a shared persistent USDA catalogue, manual cleanup/storage budgeting, and 10 USDA searches per user per day are deferred to Phase 3. See [Phase 3 backlog](PHASE_3_BACKLOG.md). None of these behaviours are claimed as implemented in Phase 2.
- Existing USDA search and private custom foods/recipes remain available. No additional Indian-food dataset/provider has been selected or integrated, and no paid service has been enabled. Expanded regional coverage remains a future evaluation, not a required Phase 2 decision.
- No further feature-scope or provider-licensing decision is required to accept the revised Phase 2 implementation. The verification and release evidence below records completion.

### Optional analytics views, not claimed as implemented

The original PRD gives additional analytics examples. Current views provide per-nutrient/calorie/protein trends, averages, target comparisons, meal contributions, water/weight/activity trends, and adherence history. A dedicated combined macro-distribution visualization and a separate protein-consistency score/streak view are not implemented. They are possible follow-on UI enhancements, not provider/licensing blockers or missing tracking capabilities.

### Explicitly outside the revised Phase 2 scope

- Notifications and reminders.
- Offline logging and synchronization; installable PWA features.
- AI, label/image recognition, voice logging, external-agent/MCP functionality.
- Backup, restore, and recovery implementation. Only the agreed Terms limitation was added.
- Third-party behaviour analytics, message broker, or a separate analytics worker.

### Verification and release tasks

- Frontend production build passes.
- Five focused frontend model tests pass, including null paused-schedule time lists and historical target selection.
- Core and API Go tests pass in Linux containers against isolated local MongoDB test databases.
- REST smoke checks cover recipe portions/overrides/snapshots, saved meals/servings, wellness conversion/CRUD/goals, and versioned scheduling/history.
- Contract checker passes for the existing OpenAPI subset and protobuf descriptors; the Markdown API design documents the additive Phase 2 routes. Full formalization of the older route inventory in OpenAPI remains documentation debt, not a new runtime feature.
- Chrome checks cover the principal new flows and narrow-screen layouts. The null paused-schedule crash found during QA was fixed in the UI and API and covered by regression tests.
- Local smoke used isolated synthetic fixtures. Authorized production Chrome sanity used a designated test account, with synthetic wellness, recipe, schedule and food records retained there; no personal-account data was modified.

Release promotion and production sanity are complete. PRs 14/15 promoted the initial Phase 2 changes; PRs 16/17 promoted the production-QA nutrient-key fix. No new domain or VPS is required for the current implementation. A future provider choice could introduce a separately approved subscription cost.

Production QA found that the old nutrient-key regex rejected B6/B12. The corrected rule is `^[A-Z][A-Z0-9_]{0,39}$`, retaining protection against unsafe MongoDB field names. Regressions cover known/unknown vitamin fields and decimal portion scaling.

Production sanity covered USDA search; wellness unit conversion/edit/goal; monthly schedule actions, adherence and versioned pause; recipe fractional portions/override review; unchanged historical nutrition after recipe edits; and analysis known-day/unknown handling. The earlier null paused-time-list crash is also regression-tested.

Evidence: [production deployment run](https://github.com/architsangal/AteThat/actions/runs/38879140661), [production QA report](https://github.com/architsangal/AteThat/pull/17#issuecomment-6212300338), [release verification](https://github.com/architsangal/AteThat/pull/15#issuecomment-6212300886). Temporary local screenshots are not durable release artifacts. The tag is a code checkpoint, not a database backup.

## Phase 3 — next

[Catalogue/quotas and original AI, voice, export, agent roadmap](PHASE_3_BACKLOG.md) are not implemented. No mandatory deliverable remains in revised Phase 2.

## Phase 4 and later — deferred

See [platform backlog](PHASE_4_BACKLOG.md). Recovery remains excluded; barcode is removed, not deferred.
