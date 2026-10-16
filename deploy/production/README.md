# AteThat production deployment

This directory deploys Caddy, the public-facing API and private Core. The frontend is separately hosted as Cloudflare Workers static assets at `atethat.architsangal.in`.

## Safety boundaries

- Caddy is the only container with host ports (`80` and `443`).
- API is reachable only from Caddy; Core is reachable only from API.
- MongoDB remains on Atlas and is never exposed from the VPS.
- `deploy/production/.env` is intentionally ignored by Git. Create it on the VPS from `.env.example`, fill it there, and set `chmod 600 deploy/production/.env`.
- The Compose memory limits are starting guardrails, not capacity guarantees. Monitor before adding projects.

## Phase 1 — initial VPS provisioning

The production stack is already deployed. The commands below are for a new/replacement server, not a routine update; do not re-clone or overwrite an existing environment file.

```sh
sudo mkdir -p /srv/atethat
sudo chown archit:archit /srv/atethat
git clone git@github.com:architsangal/AteThat.git /srv/atethat
cd /srv/atethat
git checkout main
cp deploy/production/.env.example deploy/production/.env
chmod 600 deploy/production/.env
# edit deploy/production/.env locally on the server; never paste it into chat or Git
docker compose -f deploy/production/compose.yaml --env-file deploy/production/.env up -d --build
docker compose -f deploy/production/compose.yaml ps
```

Before the first `up`, make the Cloudflare `api.atethat` DNS record point to the VPS (`46.202.167.196`) and leave it **DNS-only** until Caddy has obtained its certificate and the health check succeeds. Then it may be proxied through Cloudflare.

## Verification

```sh
curl --fail --show-error https://api.atethat.architsangal.in/healthz
docker compose -f deploy/production/compose.yaml logs --tail=100 caddy api core
```

Expect `{"status":"ok"}` from the health endpoint. Do not continue with OAuth, SMTP, or a frontend launch until this check passes.

## Deployment automation

The backend workflow runs automatically on push to `main`, not on `dev` or manual dispatch. It invokes only the restricted VPS user's forced deployment command; the server fetches `origin/main` and rebuilds the Compose stack. See [deployment/security setup](../../docs/DEPLOYMENT.md). Cloudflare independently deploys frontend production from `main`; previews use `wrangler preview` with the connected Worker name unchanged.

## Phase 2 — released production checkpoint

[v2.0.0](https://github.com/architsangal/AteThat/releases/tag/v2.0.0) pins `4e649a2a2ca1ba23395f98f415d71ed6d7a6bd03`. Backend health and authorized Chrome test-account sanity completed. No extra service was needed; see [verification](../../docs/PHASE_2_STATUS.md). A tag is not a database backup or automatic data rollback.

## Phase 3 — planned deployment requirements

Catalogue work reuses Atlas with a separate logical food database, measured storage/index budget and approved migration/configuration. No Redis/new VPS. AI/export/agent work needs explicit provider cost/privacy/retention and grant/audit decisions before rollout. Locally test, feature → `dev` → `main`, then authorized production sanity.

## Phase 4 and later — conditional operations

Broader workers/audit/analytics, notifications, offline/PWA and scaling remain [future work](../../docs/PHASE_4_BACKLOG.md). Recovery is not promised. Never assume rolling code back restores deleted or changed MongoDB data.
