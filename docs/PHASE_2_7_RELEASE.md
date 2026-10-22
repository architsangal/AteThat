# Phase 2.7 — combined nutrition and health item analysis

## Scope

- Analysis now has one Timeline above Nutrition analysis. **Explore** applies the chosen inclusive range (up to 366 Tracking Days) to nutrition and health charts. Changed-but-unapplied dates are explicitly marked; sections load independently and do not display partial results as complete.
- Health item analysis sits before Text Summary. The settings/plus picker selects up to 12 tasks, medications or supplements, including paused items for historical review. Preferences are stored on the account, not browser-local storage. Removing a chart does not modify health records or schedules.
- Each item has daily stacked status counts, an optional daily completion-percentage view, totals, and an expandable accessible table. Items are keyed by stable item ID, not name. No due schedule means no completion percentage, not failure or zero adherence. Completion excludes future occurrences. Missed and deliberate skips remain visibly distinct in analysis.
- Health events use effective scheduled instants, grouped by the account's current timezone and daily reset, using the same wall-clock rule as Core Tracking Day derivation. A rescheduled occurrence counts on its new day, once. Nutrition retains historically saved Tracking Days; the page explains this distinction rather than reassigning old food records.
- Long health ranges are read in bounded 28-civil-day chunks with full pagination and deduplication. The extra final civil date covers the following morning before reset. Stale requests are ignored; incomplete/looping/overlarge results fail visibly instead of showing misleading partial charts.
- Direct **Reschedule** replaces the one-option More actions disclosure. Completed occurrences still cannot be rescheduled.

## Contracts and storage

`GET/PATCH /preferences` adds `analysis_health_item_ids`, an optional replacement list on PATCH (empty clears it; omission preserves it). Core validates at most 12 unique canonical IDs owned by the authenticated account. Existing preference version conflicts apply, and unrelated preference saves preserve the selection. The list is stored in existing `user_preferences`; no new service, collection or destructive migration is needed.

## Verification

- 47 frontend regression tests and the TypeScript/Vite production build passed locally, including a proportional-chart layout regression. Chart drawing measures the available panel width; labels are never stretched with a non-uniform SVG transform.
- Full API and Core suites passed with an isolated MongoDB 7 replica set, including ownership, version conflicts, selection persistence and clearing without modifying schedules.
- Model tests cover reset boundaries, DST fallback, item identity, duplicates, effective rescheduling, future exclusions, full pagination and 366-day chunking.
- Chrome local checks covered picker save/reload, 7/90-day Explore, both chart modes, daily values and 390px mobile layout. Disposable local QA records were used; production records were not changed during local testing.

Actual release commit, required CI and production checks are recorded in published GitHub v2.7.0 release notes after promotion through dev and main.
