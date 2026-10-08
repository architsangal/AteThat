# AteThat

Nutrition and health tracking application. The repository is a monorepo with independent frontend, API, and Core components. The current implementation starts with the React frontend in [`apps/web`](apps/web).

## Design authority

The Phase 1 product, API, and database behavior is specified in [`docs/`](docs). The exported Google Stitch files in `stitch_atethat_nutrition_health_tracker/` are visual references for layout, typography, colour, spacing, and light/dark treatments. Their sample copy, mock values, scripted actions, and suggested features are not product requirements.

## Development

From the repository root:

```sh
npm install
npm run dev:web
npm run build:web
```

Copy `apps/web/.env.example` to `apps/web/.env.local` when a local API exists. Only public configuration may use the `VITE_` prefix. Keep credentials and tokens out of frontend source, browser storage, logs, and Git.

The first executable API and gRPC contract slice is in [`contracts/`](contracts). Check it with `make contracts-check`. Create local credentials with `make local-config`, start the stack with `make local-run`, inspect with `make local-status`, and stop with `make local-down`. This requires a running Docker daemon. See [`deploy/local/README.md`](deploy/local/README.md).

## Coding practices

- Use TypeScript strict mode and keep components small and focused.
- Place frontend-only helpers within `apps/web`; each future service owns its own helper code. Do not create a cross-service runtime `shared` folder.
- Extend the public REST and private gRPC contracts in their respective contract folders alongside each backend vertical slice.
- Keep every API request and response aligned with `docs/5. API_DESIGN.md`.
- Use semantic HTML, accessible names, keyboard focus states, and responsive layouts.
- Treat `UNKNOWN` nutrition values differently from an amount of zero.
- Store and exchange instants as UTC; format timestamps with an explicit user-selected timezone.
- Never invent successful mutations in the UI. Show errors or empty states until the backend operation succeeds.
- Run the relevant build and typecheck before committing.

## Current frontend scope

The first slice provides responsive navigation, theme switching, a landing page, authentication forms, and layouts for the Phase 1 sections. Email registration/verification, login, reset, Google sign-in (when configured), session refresh/logout, and profile/timezone settings now have API-backed paths. Food, nutrition, meal, and health screens remain empty states. Backend identity still needs live integration verification and remaining Phase 1 security/account-lifecycle work.
