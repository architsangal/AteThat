# Phase 2.6 — task occurrence and date/time UX

Release candidate, 22 October 2026. This extends v2.5.0; existing Phase 1–2.5 features remain supported. Promotion follows feature → dev → main, then verified production deployment and v2.6.0 publication.

## Behaviour

- Health occurrences default to **today in the account timezone**, independently of the nutrition reset. Shared From/Through calendar controls apply to Active, Done, Missed and All tabs, with counts across the complete paginated range. Ranges include both selected dates, up to 31 calendar days; UTC boundaries account for DST.
- Active means pending; Done includes tasks completed and medication/supplements taken. Deliberately skipped occurrences appear under Missed with “Skipped by you” detail. Enabling/disabling a recurring health item is separate from occurrence status.
- Reschedule is an action, not a tab. An unfinished occurrence moved to a future instant reappears as Active in its **new** date range, not the original one. Stable identity and schedule version are retained. Subsequent completion/skip keeps that moved date. Completed occurrences cannot be rescheduled.
- Effective schedule ordering and pagination happen after moved-date projection. Legacy RESCHEDULED records are interpreted on read without a destructive migration. Recurring schedules are never shifted by one-off rescheduling.
- Every successful action atomically snapshots before/after state in owner-scoped `health_item_event_history`. Current events carry a version; the UI sends it to reject stale actions. Account purge includes the new audit collection. The older action-history endpoint remains an original-schedule audit view; the primary occurrence tabs use effective dates.
- Native setup DOB and health-date popups are replaced with the app calendar. Calendars used inside modal dialogs remain in the dialog’s top layer. Optional dates can be cleared. Schedule times, reset time and rescheduling use labelled 24-hour hour/minute selectors with canonical HH:mm values, not native browser time spinners. Duplicate schedule times are rejected locally.
- Completed rows do not repeat a primary Done action. Routine management and extended audit statistics are secondary disclosures. Keyboard tab navigation and responsive layouts are supported.

## Verification and promotion

- All 37 frontend regression tests, TypeScript/Vite production build and contract validation passed.
- Full API and Core Go suites passed against an isolated MongoDB 7 replica set, including effective-date pagination, stable identity, owner isolation, stale-action conflicts, completion rules and transactional history.
- Local HTTP acceptance passed: complete/skip, reschedule out of the original range into the new range, subsequent completion, stale version rejection and rejection of completed-task moves.
- Chrome extension verification covered setup DOB year/month selection, all four occurrence tabs, date range changes, rescheduling with a calendar inside a modal, completion after rescheduling, and responsive 390px layout. A task with two 24-hour times saved correctly; incomplete time selection prevented saving.
- Docker Desktop was unresponsive during local QA. Equivalent isolated native API/Core services and official MongoDB 7 were used instead; no production or existing local account records were used.

GitHub release notes record the actual deployed commit, hosted CI and production checks. This source document does not claim production checks before they occur.

## Phase 3

No new AI, barcode, notification, offline/PWA, provider expansion or recovery functionality. No new infrastructure service. A code tag is not a database backup.
