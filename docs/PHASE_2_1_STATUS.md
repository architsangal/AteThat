# Phase 2.1 — everyday UX improvements

Status: implemented and verified locally on `fix/phase-2.1-ux`, 20 October 2026. This is a local validation milestone, not a published release or production deployment.

Phase 2.1 began with the 22 findings in the owner-reviewed UX audit. The first pass was incomplete: native browser prompts/validation and inconsistent hover/focus states remained. The refinement below addresses the owner's screenshots and adds the newly requested search accounting and welcome guide. Nutrition calculations, explicit confirmation and idempotent writes remain in place. Backend contracts now include search usage and account-persisted onboarding.

See the [individual feature checklist](PHASE_2_1_FEATURE_CHECKLIST.md) for a user-facing inventory and review steps.

## Refinement after owner review

- Added generic `TASK` health items for routines such as brushing teeth or applying a solution. They use names, optional instructions/notes and the existing scheduling/history lifecycle, without dosage fields. Tasks display Done/done rather than Taken/taken; the compatible `TAKEN` wire status still drives completion statistics. Backend validation and isolated lifecycle/ownership tests cover the new type. This remains local work pending authorized release.

- Replaced every application `window.prompt`/`window.confirm` with themed dialogs, including correction amount/reason, rename, portion labels, removals and account-deletion confirmation. Rescheduling uses a labelled datetime control instead of an ISO-string prompt.
- All React forms use shared inline validation with required, minimum/maximum, length and input-type checks. Invalid controls are focused, and collapsed ancestor sections open when necessary. Search has one outer focus ring; Add no longer transforms or shades its whole result row.
- Shared dialogs render in the document body so panel layout/last-child rules cannot displace their position. Dialogs contain focus, close with Escape when idle and restore the opener.
- History correction uses a single amount/reason form followed by a nutrition-review dialog. Confirmation updates the visible meal and daily totals.
- Local database search is the default in logging and recipe ingredient selection. Explicit USDA discovery imports normalized public source records into a separate logical Mongo database (`<application_database>_foods`). Records are deduplicated by FDC ID; private foods stay in their account-owned collection and dedicated tab.
- Backend-atomic allowances are 10 successful USDA searches per authenticated account per UTC day. Successful empty pages count; failed provider/decode requests refund their reservation. Every next-page discovery counts. Local searches remain unlimited and are counted separately. Read-only status requests do not consume searches. The UI shows database size, result count, account usage and the reset time. HTTP 429 identifies exhaustion.
- Welcome is a three-step, skippable profile-and-feature guide. Date of birth, height, weight and goal are optional. Completion/skip is stored in the profile and survives sign-ins/devices. Existing accounts without the flag see the guide once. Header help and Settings allow replay. Optional measurements and tracking goal can also be edited later under Settings → Optional profile details.
- Tablets through 1100px use the single-column logger, drawer and persistent Review bar. Laptop baskets are compact enough to show Review at 1366×768.

The broader Phase 3 global provider budget/cooldown, storage-budget enforcement, scheduled catalogue refresh and maintenance remain backlog work. No new subscription or paid provider was introduced.

## User experience

The normal food path is **Search → Add → Review → Log food**. Search appears first. Laptop users can edit amounts in a sticky selection panel; phone and tablet users get a persistent Review action and a separate Edit selected foods control. Review presents the foods, amounts, energy and main macros. Optional nutrient edits, favourites, household portions and reusable-meal actions are disclosed only when requested.

Unconfirmed food selections, meal category, name, mapped portions and overrides survive navigation within the signed-in app. They stay in memory and are cleared on sign-out/session expiry or a page reload; private draft data is not written to local storage. Expired review tokens are never reused after navigation: review is requested again from the retained selection.

## Audit traceability

| Finding | Implementation |
|---|---|
| UX-01 Search buried | Search/source tabs are first; saved-food tools use one collapsed disclosure. |
| UX-02 Automatic override form | Nutrition editing is closed by default, attached to each selected food. |
| UX-03 Review far below results | Direct persistent mobile/tablet Review; independently scrollable sticky laptop basket. |
| UX-04 Long technical review | Shared concise nutrition summary; readable labels; further known nutrients disclosed; missing values remain unknown. |
| UX-05 Ambiguous results | Generic reference foods rank before context-free packaged products; preparation/query matches influence ranking; product type, portion basis, energy and expandable source ID/details distinguish results. Different source IDs are preserved; repeat IDs from pagination are deduplicated. |
| UX-06 Draft lost on navigation | Shared session-scoped draft provider; guards prevent completed requests from restoring a draft after the page has unmounted. |
| UX-07 Hidden recipe logging form | Portion entry and its review open in a focused native dialog. |
| UX-08 Old route scroll retained | Route changes reset scroll to the top and focus the page heading. |
| UX-09 Conflicting library names | Navigation/page title both use Foods & meals; ordinary saved meals and yielded recipes have distinct collections. |
| UX-10 Always-open creation forms | Food, recipe and health creation/edit forms open on request. |
| UX-11 Water entry buried | Logging first, common amounts, optional time/notes and goal, collapsed history/trends. |
| UX-12 Conflicting time contexts | Review displays the account timezone; wellness entry and record timestamps consistently use the labelled device timezone; day-boundary context is explained in history; UTC health filters are labelled; health creation waits for the actual profile timezone. |
| UX-13 Text schedule formats | Repeated native time inputs, Add another time, selectable month dates, conditional weekdays and advanced recurrence disclosure. Shared by schedule creation/editing. |
| UX-14 Mobile horizontal tables | Nutrient and daily tables become labelled stacked rows. Also applied to recipe comparisons, wellness and health history. |
| UX-15 Tablet sidebar squeeze | Drawer and bottom navigation through 960 px; larger viewports retain the sidebar. |
| UX-16 Scattered food actions | Portion and amount stay with each selection; other actions live in its More options menu. |
| UX-17 Gram-only household entry | Existing owned serving mappings appear in the selected food's Portion selector, with an explicit gram/ml equivalent and a serving count. Users can save a weighed household portion from the same food. |
| UX-18 False empty/zero loading | Explicit loading/error states for the dashboard, collections, source tabs, food history, wellness and settings; schedule history suppresses provisional counts. |
| UX-19 Target setup detour | Direct Complete profile and return path; profile save returns to Targets; unused proposal/history panels are hidden. |
| UX-20 Active/paused ambiguity | Separate Item enabled/disabled and Schedule paused/frequency labels. |
| UX-21 Dashboard clutter | Smaller energy panel, progress meters only when a target exists, fewer repeated logging links, profile prioritised in settings. |
| UX-22 Technical copy/counts | Selected foods/meal/review language, readable enum choices, singular/plural counts in the changed flows; technical nutrient codes removed from routine confirmation. |

USDA brand names and automatic household weights are not present in the current API. This change uses available catalogue metadata and owned weight mappings rather than inventing either. Ranking applies to the fetched results, not the entire provider catalogue. Further catalogue enrichment belongs to Phase 3.

## Local verification

Environment: Vite at `http://localhost:5173`, local API at port 18080, local Mongo replica set and Mailpit. No production account or production records were used for Phase 2.1 testing.

Passed automated checks:

- Frontend TypeScript and production build.
- Eight model tests: unknown versus zero, averages, date boundaries, historical targets, null paused-schedule times, source-result ranking and pagination identity.
- Existing localhost-only REST smoke: recipe yields/fractional portions, saved meals, serving ownership, overrides, unknown/zero nutrients, idempotent confirmation, preserved historical nutrition, wellness CRUD/conversions/goals, monthly schedules, versioned pauses and schedule history.
- OpenAPI/protobuf validation and whitespace checks.

Passed browser checks:

- Real local USDA banana search; raw reference foods appear before packaged BANANA results.
- Direct mobile review and optional selection editing; amount changes and return from review.
- A mapped 1.5 × 100 g bowl survives library navigation, previews as 150 g / 300 kcal, saves, and appears in history.
- Zero energy override previews as 0 kcal. Changing amount to 50 g clears it and restores source-derived 48.5 kcal.
- Recipe portion dialog opens immediately; 0.5 serving previews as 18.75 g / 37.5 kcal and saves.
- A 500 ml water record saves from the primary entry flow.
- Monthly schedule with dates 1 and 15 and times 08:00/20:00 saves; both times appear when reopened.
- Scrolled schedule editor → Analysis navigation returns to scroll position zero.
- Fresh local account email login, missing-DOB guidance, profile completion/automatic return, benchmark review and acceptance.
- Food layout at 360×800, 390×844, 768×1024, 1024×768, 1366×768 and 1440×900. Search remains visible; the correct Review control is available; no document-wide horizontal overflow.
- Laptop selection remains at the top of the viewport while the result list scrolls.
- Mobile analysis tables show labelled values without internal horizontal overflow.
- Dark and light appearances sampled, with screenshots retained outside the repository in the workspace's `phase-2.1-evidence` folder.

Browser tests use disposable synthetic local accounts and retain their records for owner inspection. Actual iOS/Android keyboards, Safari and assistive-technology testing remain outside this desktop-browser pass. The refinement changes Core/API Go source and generated contracts; both services were rebuilt locally. See the refinement validation below.

## Review locally

Keep Docker running. From the repository root, start the local stack with `make local-run` if needed, then `npm run dev --workspace @atethat/web -- --host localhost`. Open `http://localhost:5173`.

The existing fixture `phase2-local-test@example.invalid` and its synthetic password are defined in `scripts/phase-two-local-smoke.mjs`; running that script targets localhost only. The browser walkthrough also created a separate disposable `phase21-local-test@example.invalid` account for first-time setup checks. No credentials are committed in this status document.


## Refinement validation

- Web production build and TypeScript checks pass.
- Eight existing/new frontend model tests pass; OpenAPI and protobuf descriptor validation pass.
- Both Go suites pass with `go test -ldflags=-linkmode=external ./...`. The ordinary macOS internal linker produced test binaries rejected by the local loader; using the system linker resolved that environment issue. macOS SDK-version warnings remain nonfatal.
- `scripts/phase-two-local-smoke.mjs --trends` passes recipe portions, unknown/zero nutrition, overrides, idempotent confirmation, history preservation, servings, saved meals, wellness conversions/CRUD/goals and schedule versions.
- `scripts/phase-2.1-local-smoke.mjs` passes persisted onboarding, shared catalogue import/deduplication, same-name distinct IDs, matching counts, per-account counters, atomic parallel quota exhaustion, HTTP 429, failed-provider refunds, successful empty discovery, local availability after exhaustion UTC day isolation, unauthenticated status rejection and private-food isolation. It uses cached synthetic provider payloads and disposable accounts only against localhost.
- Browser: saved profile → both guide steps → food logging; replay/skip; empty and one-character inline validation; USDA discovery → local lookup with unchanged allowance; add → review → confirm; correction to 150g/168kcal → updated daily total; phone edit and nested portion-name dialog; recipe local ingredient search with shared allowance; tablet drawer; light/dark sampling.
- Food logging checked at 360×800, 390×844, 768×1024, 1024×768, 1366×768 and 1440×900. The refined tablet breakpoint was checked again at 1024px; laptop Review was measured within the 768px viewport. No document-wide horizontal overflow was observed.
- Evidence is in the workspace's `phase-2.1-evidence/refined-*.jpg` files. Browser-extension message-channel errors were present; no application crash was observed. This is desktop Chrome viewport testing, not a claim of physical-device/Safari or assistive-technology certification.

Work remains local and uncommitted; nothing was pushed or deployed by this task.

Local mail note: inherited SMTP authentication settings interrupted the final fresh-account fixture registration. A temporary `/tmp/atethat-local-test-mail.yaml` Compose overlay points Core to the existing local Mailpit service with empty local SMTP credentials. No external mail credentials were changed or printed, and application SMTP security was not changed. Ordinary Compose startup may restore the inherited environment; local test mail should use Mailpit's no-auth configuration.

## USDA provider-limit diagnosis after review

The owner's paneer search failed because USDA returned HTTP 429 from inside the local Core container using the shared `DEMO_KEY`. This is independent of AteThat's 10-per-account daily allowance. The API previously hid this distinction behind `CORE_UNAVAILABLE`.

Core now distinguishes provider throttling and rejected provider credentials; API returns an actionable provider-specific message. Both server regression suites pass. A live local fixture reproduced `USDA_PROVIDER_RATE_LIMITED` after rebuild and retained its allowance at 10 before/10 after. The message fix does not remove the upstream restriction: local search remains available, while upstream access requires provider recovery or configuring a USDA key instead of the shared demo connection.
