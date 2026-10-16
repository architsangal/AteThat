# Local development dependencies

`make local-up` starts MongoDB 7 as a single-node replica set and Mailpit. The replica set is needed for atomic food-log transactions used by the implemented Core service. `make local-status` shows container status; `make local-down` stops containers without deleting the MongoDB volume. Mailpit's inbox is at `http://localhost:8025`; SMTP is at `localhost:1025`.

Run `make local-config` once to create ignored, permission-restricted development credentials in `deploy/local/.env`, then `make local-run` to build and start API and Core alongside the dependencies. Run the web app separately using `npm run dev:web` and open `http://localhost:5173` (use `localhost`, not `127.0.0.1`, for the configured local origin). Vite proxies `/api/v1` to `http://localhost:18080`, keeping local auth cookies on the same browser origin. Local API is also directly available at `http://localhost:18080/api/v1`; host port 8080 was already used by another local process.

Google login is optional in local development until you obtain OAuth credentials. To enable it, add `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, and `GOOGLE_REDIRECT_URI=http://localhost:18080/api/v1/auth/google/callback` to `deploy/local/.env`, and set `VITE_GOOGLE_SIGNIN_ENABLED=true` in `apps/web/.env.local`. Configure the same redirect URI in Google's OAuth client. New Google users start with `UTC` and are sent to profile settings to choose their default timezone manually. Existing email/password accounts are never silently merged with Google identities; an authenticated user can explicitly link Google from settings.

Keep `TRUSTED_PROXY_CIDRS` empty for direct local access. When API traffic is routed through Caddy, set it to Caddy's private container/network CIDR (for example `172.18.0.0/16`) and configure Caddy to set `X-Forwarded-For` from the connecting client. The API honors that header only from this trusted direct peer, so login rate limits stay per client and cannot be spoofed by public callers.

MongoDB has no authentication in this local-only stack. Its exposed port is bound to loopback. Never reuse this Compose file as a public deployment or open its MongoDB port to a network. For production, Atlas or a separately secured self-hosted replica set is configured through the Core service's MongoDB URI.

To test real Gmail delivery locally, enable two-step verification on the project Gmail account and create a Gmail app password. In ignored `deploy/local/.env`, set `SMTP_HOST=smtp.gmail.com`, `SMTP_PORT=587`, `SMTP_FROM` and `SMTP_USERNAME` to that Gmail address, and `SMTP_PASSWORD` to the app password. Do not commit or share the app password. Re-run `make local-run`, then use a password-reset or verification flow for an existing account at an inbox you control. Core logs delivery failures without logging the recipient or token.

The `mongo-init` service runs `rs.initiate` only when needed; on a reused volume it leaves the existing replica set intact. A host application can connect with `mongodb://localhost:27017/atethat_dev?replicaSet=rs0&directConnection=true`. Inside Compose, use `mongo:27017` instead of `localhost`.

## Phase 1 — local baseline

Mongo replica set, Mailpit, API/Core and separately run Vite support isolated authentication, food logging and schedules. This environment is never the production stack.

## Phase 2 — local verification

Run frontend typecheck/build/model tests and API/Core tests with an isolated Mongo test URI. `node scripts/phase-two-local-smoke.mjs` deliberately uses localhost and its synthetic account only. It must not be repointed at production. See [release verification](../../docs/PHASE_2_STATUS.md).

## Phase 3 — planned setup

Reuse local Mongo with a separate logical food database when the catalogue is implemented. Add quota/provider failure fixtures, storage-budget tests and safe cleanup tests. AI/export/agent work needs mocked providers and permission/retention tests before any external processing. No Redis/new paid dependency is required for catalogue work.

## Phase 4 and later — conditional setup

Only add workers/brokers/notification tooling when an approved feature needs them. Offline/PWA and recovery are not implemented by this local stack.
