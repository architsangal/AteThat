# AteThat web app

React, TypeScript, and Vite frontend. The Stitch export at the repository root is a visual reference for layout, typography, spacing, and light/dark colour treatment. Product features and API behaviour come from `docs/3. SYSTEM_DESIGN.md` and `docs/5. API_DESIGN.md`.

Run `npm install` from the repository root, then `npm run dev:web`. Set `VITE_API_BASE_URL` in `apps/web/.env.local` when the API Service is available. The public client must never contain secrets.

## Phase 1 — released

Responsive shell/themes, authentication, profile, FDA general benchmarks, food search/custom foods/servings, reviewed grouped logging, meals, dashboard/history and basic schedules are API-backed. Never simulate successful writes.

## Phase 2 — released

Expanded nutrient analysis, recipes/portions/overrides, water/weight/activity/optional water goal, advanced schedule history/adherence and DOB month/year selection. [Release verification](../../docs/PHASE_2_STATUS.md) covers production test-account sanity.

## Phase 3 — planned

Local-first food search plus explicit quota-limited USDA discovery; assisted label/photo/text/voice proposals; exports and agent permissions UI where needed. Preserve unknown values and explicit review. Provider/privacy/sequence decisions remain open; see [backlog](../../docs/PHASE_3_BACKLOG.md).

## Phase 4 and later — deferred

Notifications and offline/PWA are not current capabilities. Barcode is removed. See [platform backlog](../../docs/PHASE_4_BACKLOG.md).

## Build configuration and verification

Production runs as Cloudflare Workers static assets. `VITE_` values are public compile-time configuration, not runtime secrets. Build/typecheck from the root; run the nutrition and wellness schedule model tests. Change configuration through a rebuild; do not add Worker runtime code solely to hold frontend environment values.
