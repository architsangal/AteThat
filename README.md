# AteThat

Nutrition and health tracking application. The repository is a monorepo with independent frontend, API, and Core components. Phases 1 and 2 are released; the frontend is in [`apps/web`](apps/web), with Go API/Core services. Start with the [documentation index](docs/README.md).

## Design authority

The phase-specific product, API, and database behavior is specified in [`docs/`](docs). The exported Google Stitch files in `stitch_atethat_nutrition_health_tracker/` are visual references for layout, typography, colour, spacing, and light/dark treatments. Their sample copy, mock values, scripted actions, and suggested features are not product requirements.

## Development

From the repository root:

```sh
npm install
npm run dev:web
npm run build:web
```

Copy `apps/web/.env.example` to `apps/web/.env.local` for local API configuration. Only public configuration may use the `VITE_` prefix. Keep credentials and tokens out of frontend source, browser storage, logs, and Git.

Executable API and gRPC contracts is in [`contracts/`](contracts). Check it with `make contracts-check`. Create local credentials with `make local-config`, start the stack with `make local-run`, inspect with `make local-status`, and stop with `make local-down`. This requires a running Docker daemon. See [`deploy/local/README.md`](deploy/local/README.md).

## Coding practices

- Use TypeScript strict mode and keep components small and focused.
- Place frontend-only helpers within `apps/web`; each service owns its own helper code. Do not create a cross-service runtime `shared` folder.
- Extend the public REST and private gRPC contracts in their respective contract folders alongside each backend vertical slice.
- Keep every API request and response aligned with `docs/5. API_DESIGN.md`.
- Use semantic HTML, accessible names, keyboard focus states, and responsive layouts.
- Treat `UNKNOWN` nutrition values differently from an amount of zero.
- Store and exchange instants as UTC; format timestamps with an explicit user-selected timezone.
- Never invent successful mutations in the UI. Show errors or empty states until the backend operation succeeds.
- Run the relevant build and typecheck before committing.

## Phase 1 — released baseline

The local stack has API-backed email registration/verification, login/reset, Google sign-in and linking hooks, profile/timezone and daily reset settings, FDA general nutrition benchmarks, USDA search, custom foods, named servings, grouped food logging, corrections, dashboard, nutrition history/analysis, favourites, recent/frequent foods, reusable meals, and medication/supplement schedules and actions. The UI is responsive and supports light/dark themes.

USDA uses `DEMO_KEY` by default. Set `USDA_API_KEY` in `deploy/local/.env` to use your own key. Google sign-in needs environment-specific OAuth credentials and an exact registered callback; production sign-in has been verified. Email in local development is captured by Mailpit. Account deletion is a verified request that immediately signs out all sessions, permits verified cancellation for 30 days, then is purged by Core's hourly worker. The local stack is isolated from the separately deployed production system. See `deploy/local/README.md` for startup and local URLs.

## Phase 2 — released as v2.0.0

Expanded nutrient analysis/recipes, water/weight/activity logging, optional personal water goal, advanced monthly/multi-time/versioned schedules, DOB month/year picker and supporting fixes are deployed. See [release status and verification](docs/PHASE_2_STATUS.md) and [v2.0.0](https://github.com/architsangal/AteThat/releases/tag/v2.0.0). A release tag preserves code, not database recovery.

## Phase 3 — planned

Local-first MongoDB food catalogue, explicit USDA discovery capped at 10 searches/user/day, bounded storage and manual cleanup are agreed next-phase work. Original AI label/photo/text/voice assistance, CSV/JSON exports and scoped external-agent/MCP capabilities remain planned, with provider/privacy/cost decisions open. See [Phase 3 backlog](docs/PHASE_3_BACKLOG.md).

## Phase 4 and later — conditional

Notifications, offline/PWA, broader worker/audit/analytics and scaling remain [future platform work](docs/PHASE_4_BACKLOG.md). Recovery needs separate approval. Barcode is removed entirely; no extra Indian-food provider is selected.

## Contribution and deployment flow

Feature branches start from `dev`; locally verify, PR into `dev`, then promote `dev` to `main`. Backend deploys automatically on push to `main` through restricted VPS SSH; Cloudflare builds the static Worker independently. Previews use `wrangler preview`, not a renamed production Worker. See [deployment](docs/DEPLOYMENT.md). Do not push/merge/deploy without task authorization.
