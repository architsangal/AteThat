# AteThat production deployment

This directory deploys only the public API. The frontend is a separate Cloudflare Pages project at `atethat.architsangal.in`.

## Safety boundaries

- Caddy is the only container with host ports (`80` and `443`).
- API is reachable only from Caddy; Core is reachable only from API.
- MongoDB remains on Atlas and is never exposed from the VPS.
- `deploy/production/.env` is intentionally ignored by Git. Create it on the VPS from `.env.example`, fill it there, and set `chmod 600 deploy/production/.env`.
- The Compose memory limits are starting guardrails, not capacity guarantees. Monitor before adding projects.

## First deployment on the VPS

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

Before the first `up`, make the Cloudflare `api` DNS record point to the VPS (`46.202.167.196`) and leave it **DNS-only** until Caddy has obtained its certificate and the health check succeeds. Then it may be proxied through Cloudflare.

## Verification

```sh
curl --fail --show-error https://api.atethat.architsangal.in/healthz
docker compose -f deploy/production/compose.yaml logs --tail=100 caddy api core
```

Expect `{"status":"ok"}` from the health endpoint. Do not continue with OAuth, SMTP, or a frontend launch until this check passes.

## Deployment automation

The repository includes a manually triggered GitHub workflow. It deliberately does not deploy on every push. Before enabling it, follow the restricted-user setup in `docs/DEPLOYMENT.md`; the workflow should be allowed to run only the fixed server deployment script, never an arbitrary remote command.
