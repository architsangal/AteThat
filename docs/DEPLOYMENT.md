# AteThat - Deployment

| Field | Value |
|---|---|
| Status | Production deployed through Phase 2 / v2.0.0; reconciled 2026-10-16 |
| Scope | Hosting, domain, routing, TLS, and deployment-cost decisions |
| Product | AteThat |

## Phase 1 — deployed infrastructure baseline

### 1. Current deployment decision

The initial host is **Hostinger KVM 2** in the India region. The domain is delegated to Cloudflare, while Hostinger remains the registrar.

| Resource | Initial choice |
|---|---|
| VM | Hostinger KVM 2 — Ubuntu 26.04 LTS |
| VM resources | 2 vCPU, 8 GB RAM, 100 GB NVMe storage, 8 TB transfer |
| Region | India; do not use Malaysia for the India-first product unless the final latency test changes this decision |
| Database | MongoDB Atlas Free cluster (`M0`) initially |
| Database recovery | No application-managed backups or recovery guarantee in the initial configuration |
| Reverse proxy | Caddy |
| Containers | Docker Compose, with one isolated Compose project and private network per application |
| TLS | Caddy-managed public TLS certificates and automatic HTTP-to-HTTPS redirects; Cloudflare is set to Full (strict) after origin TLS works |
| Frontend | Cloudflare Workers serves the static frontend |
| DNS | Frontend hostname points to Cloudflare Workers static assets; API hostname points to the VM |

KVM 2 is the selected cost-first starting point for AteThat and several small, low-traffic projects. Increase capacity only after resource monitoring and load testing show that the VM is constrained. KVM 4 remains the vertical-scale option; it is not the initial choice.

### 1.1 Infrastructure state reconciled 2026-10-16

| Item | State |
|---|---|
| Domain | `architsangal.in` is registered at Hostinger and active through 2029-10-10. |
| DNS authority | Cloudflare Free zone is active; Hostinger nameservers were replaced with the assigned Cloudflare nameservers. |
| VPS | Hostinger KVM 2, Ubuntu 26.04 LTS, running in India/Mumbai. |
| Server access | Key-only SSH through non-root user `archit`; root SSH and SSH password authentication are disabled. `sudo -i` retains administrator access. |
| Firewall | UFW allows only TCP 22, 80, and 443. Fail2ban and automatic security updates are enabled. |
| Container runtime | Docker Engine/Compose run Caddy, API and private Core in `atethat-production`; repository `/srv/atethat`. |
| DNS records | `atethat.architsangal.in` serves the frontend; `api.atethat.architsangal.in` routes to the VPS/Caddy. Manage authoritative DNS in Cloudflare, not Hostinger. |

The VPS public address is operational infrastructure, not a secret. It must not be hard-coded in application source; use the product-specific Cloudflare `api.atethat` record in the parent zone, not a generic `api` hostname.

### 2. Cost record

The Hostinger 24-month KVM 2 checkout price observed during planning was:

| Item | Amount |
|---|---:|
| Base price for 24 months | ₹19,176.00 |
| GST | ₹3,451.68 |
| Total payable | ₹22,627.68 |
| Published renewal price | ₹1,199/month before GST for a 24-month renewal |

Final checkout pricing, taxes, and promotion eligibility must be checked before purchase.

### 3. Coupon record

| Field | Value |
|---|---|
| Code | `GXDGLOBAL` |
| Claimed discount | 20% off |
| Intended use | Hostinger checkout for the initial VPS purchase |
| Verification status | User supplied; apply it at checkout and confirm that the final total is lower before paying |

Use one promotion only. Do not assume a coupon changes the published renewal price unless Hostinger's checkout explicitly says so.

### 4. Domain and routing model

The parent domain is `architsangal.in`. Each project has separate public hostnames so projects do not overlap:

```text
atethat.architsangal.in       -> Cloudflare Worker static assets (AteThat frontend)
api.atethat.architsangal.in   -> Hostinger KVM / Caddy / AteThat API Service

rollercoaster.architsangal.in     -> RollerCoaster frontend (future)
api.rollercoaster.architsangal.in -> RollerCoaster API (future)
```

The domain is deployment configuration, not application code. Caddy is the only service listening publicly on ports 80 and 443. It routes each recognized hostname only to its matching project; unknown hostnames are rejected.

### 5. Initial VM layout

```text
Hostinger KVM 2
├── Caddy (shared public edge)
├── ate-that/                 # Docker Compose project; private app network
│   ├── api
│   └── core
├── rollercoaster/            # Separate future Compose project and network
└── other-projects/           # Each project remains independently deployed
```

MongoDB Atlas is external to the VM. The VM must not expose MongoDB port `27017`.

### 6. Deployment checklist and remaining operational checks

- [x] Purchase the parent domain and keep registrar access secure.
- [x] Select the India VM region and install Ubuntu 26.04 LTS.
- [x] Configure key-only SSH, firewall, Fail2ban, Docker, and automatic security updates.
- [x] Delegate nameservers to Cloudflare.
- [x] Configure `api.atethat` for the VPS and the Worker custom frontend hostname.
- [x] Connect `atethat.architsangal.in` to the static Worker and configure the public API base URL.
- [x] Add versioned Caddy, Docker Compose, health-check, and log-rotation configuration to the repository.
- [ ] Operational recheck: verify Atlas IP allowlist matches intended production/local access; this reconciliation did not audit the live allowlist.
- [x] Store production secrets in the ignored VPS environment file; do not copy values into documentation.
- [x] Deploy containers with the unique `atethat-production` project and private network.
- [x] Verify public HTTPS, authentication/email, food logging and API/Core health in release checks.
- [ ] Record baseline CPU, memory, disk, and response-time measurements before adding further projects.

### 7. Production configuration and delivery

The production artifacts are in [`deploy/production`](../deploy/production). They have two non-negotiable properties:

1. Caddy is the only process published to the internet (`80` and `443`). API and Core have no host ports. API is on a private Docker network with Core; Core is on no public-facing Docker network.
2. The production environment file is created only on the VPS at `deploy/production/.env`; it is ignored by Git and must be mode `600`.

The server must use fresh signing material, a fresh Google OAuth client secret, and a fresh Gmail app password. Local development secrets must not be copied to production. Rotate the Google client secret and Gmail app password exposed during earlier local debugging before filling production values.

### 7.1 Branch and release policy

- `dev` is the integration branch.
- `main` is the production release branch.
- Production backend deployment runs automatically on push to `main` and deploys `origin/main`. A push to `dev` does not deploy the production backend.
- Merge a reviewed, green `dev` change into `main` for automatic production deployment.

### 7.2 GitHub Actions

`CI` runs on pull requests and pushes to `dev` or `main`. It type-checks/builds the web app, tests both Go modules, and validates the production Compose file.

`Deploy production` is triggered by push to `main` (not `workflow_dispatch`). Its SSH key must be a deployment-only key assigned to a locked-down server user with a forced command. The workflow does not receive an administrator key and cannot provide a branch name or an arbitrary shell command.

One-time GitHub repository configuration:

1. In **Settings → Environments**, create `production`; add required reviewers when a second reviewer is available.
2. Create a VPS deployment trigger key pair locally. Keep its private key only in GitHub's `VPS_DEPLOY_SSH_PRIVATE_KEY` environment secret; add its public key to the restricted server user's `authorized_keys` with a forced command.
3. Add these `production` environment secrets:
   - `VPS_DEPLOY_HOST` — the server host/IP.
   - `VPS_DEPLOY_USER` — the restricted deployment user, not `archit`.
   - `VPS_DEPLOY_SSH_PRIVATE_KEY` — the deploy-trigger private key.
   - `VPS_DEPLOY_KNOWN_HOSTS` — the pinned `ssh-keyscan -H <server>` output, verified from a trusted connection.
4. Generate a second, read-only deploy key on the VPS. Add only its **public** half to the GitHub repository as a read-only deploy key so the server can fetch `main`. This key is separate from the Actions trigger key.

Install [`deploy/production/server/atethat-deploy`](../deploy/production/server/atethat-deploy) as `/usr/local/bin/atethat-deploy` before adding the forced command. It accepts no arguments, takes a deployment lock, fetches only `origin/main`, and runs the fixed Compose deployment. This separation is deliberate: a compromised GitHub Actions secret cannot become unrestricted server shell access.

### 7.3 DNS and TLS order

Preview-build configuration reconciled 2026-10-25: use `npx wrangler versions upload` for the static-only frontend. This creates a version URL without promoting production and avoids the observed beta `wrangler preview` duplicate-name failure (100338) on repeated branch builds. Production remains `npx wrangler deploy` on `main`. Both build commands must include `VITE_API_BASE_URL=https://api.atethat.architsangal.in/api/v1` and `VITE_GOOGLE_SIGNIN_ENABLED=true`; the preview configuration previously omitted `/api/v1`. No branch protection or production routing is weakened. Named isolated-resource previews are not needed for this assets-only Worker.

1. Create `A  api.atethat  →  <VPS public IP>` in Cloudflare and initially set it to **DNS-only**.
2. Put the production `.env` on the VPS and run the first Compose deployment. Caddy then obtains its public certificate.
3. Confirm `https://api.atethat.architsangal.in/healthz` returns `{"status":"ok"}`.
4. Set Cloudflare SSL/TLS encryption to **Full (strict)**. Never use Flexible mode.
5. Change the API record to **Proxied** if desired, then later allow Cloudflare source ranges only for ports 80/443 at the origin. Do not apply that origin rule before the direct certificate/health verification is complete.
6. Deploy the frontend as a Cloudflare Worker with static assets and attach `atethat.architsangal.in`. The versioned [`wrangler.jsonc`](../wrangler.jsonc) serves `apps/web/dist` as a single-page application without executing Worker code per asset request. Its build environment must use `VITE_API_BASE_URL=https://api.atethat.architsangal.in/api/v1` and `VITE_GOOGLE_SIGNIN_ENABLED=true`.

Keep port 22 globally reachable but key-only, as explicitly chosen by the owner. Do not apply an SSH source-IP allow-list without a deliberate change in access policy.

### 8. Scaling triggers

Keep KVM 2 while it has sufficient measured headroom. Reassess the VM tier when sustained resource use, not account count alone, indicates pressure:

- Sustained memory usage reaches approximately 70-75% after normal operating headroom.
- Sustained CPU saturation, increased request latency, or swapping occurs under a representative load test.
- Additional projects require persistent databases, background workers, build workloads, or materially higher traffic.
- MongoDB Atlas M0 becomes a data-size, throughput, or connection bottleneck.

The architecture uses containers and hostname-based routing so moving from KVM 2 to KVM 4 is a deployment resize/migration, not an application rewrite.

### 9. KVM 2 container resource budget

Docker containers have no useful memory ceiling unless one is configured. Every stateful container must therefore have both an application-level limit and a Docker memory limit. The following are starting reservations for an 8 GB KVM 2, not promises about peak usage. Measure and adjust them under representative load.

| Component | Expected quiet usage | Initial configuration budget | Notes |
|---|---:|---:|---|
| Linux, Docker daemon, system services, logs, and file cache | 0.75-1.25 GB | 1.25 GB reserved | Do not allocate all VM RAM to containers. |
| Caddy | 50-100 MB | 128 MB limit | Serves static files, TLS, and reverse proxying. |
| AteThat API + Core Service combined | 150-400 MB | 768 MB combined limit | Profile real usage after implementation; Go memory grows with traffic and configured connection pools. |
| MongoDB Community (optional local database) | Variable; cache grows with use | 2 GB container limit; 1 GB WiredTiger cache | On an 8 GB host, the default WiredTiger cache calculation is about 3.5 GB and is too large for a shared VM. |
| Redis (not in the current AteThat design) | 20-50 MB plus cached data | 256 MB container limit; `maxmemory 128mb` | Redis is optional. `maxmemory` limits keys, but leave headroom for allocator and process overhead. |
| RabbitMQ (not in the current AteThat design) | 150-300 MB plus queued messages | 768 MB container limit; 512 MiB absolute memory watermark | RabbitMQ is optional. Keep queues short and consumers active; the watermark throttles publishers but is not a hard process cap. |
| Each additional small web/API project | 100-300 MB | 512 MB per project maximum initially | Give every project a measured limit and a private network. |

### 9.1 Practical KVM 2 scenarios

| Deployment shape | Approximate reserved RAM | KVM 2 assessment |
|---|---:|---|
| Caddy + AteThat services + Atlas M0 | 2.1 GB | Comfortable starting point with room for several small projects. |
| Above + local MongoDB | 4.1 GB | Viable, but reduces room for other projects; monitor closely. |
| Above + local MongoDB + Redis + RabbitMQ | 5.1 GB | Technically viable for AteThat and one or two small projects, but not the preferred layout for five or six projects. |

For AteThat Phase 1, do not add Redis or RabbitMQ unless a concrete requirement needs them. The current design uses MongoDB TTL-backed provider caching and synchronous gRPC between the API and Core services, so neither dependency is needed merely because containers are being deployed.

If multiple projects eventually require their own databases, queues, and caches on the same VM, that is a valid trigger to move from KVM 2 to KVM 4. Database, Redis, and RabbitMQ ports remain private; Caddy exposes only HTTP/HTTPS.

### 9.2 Shared-platform allocation after AteThat

Only the AteThat API and Core Service are AteThat-specific. Caddy can be shared by every project. MongoDB, Redis, and RabbitMQ can technically be shared platform services, but every project must have separate credentials and namespace boundaries (MongoDB database/user, Redis ACL/key prefix, RabbitMQ virtual host/user). Sharing them also creates a shared failure domain, so do this only for small, low-risk projects.

| Allocation on KVM 2 | RAM remaining from 8 GB |
|---|---:|
| After reserving Linux/Docker (1.25 GB), shared Caddy (128 MB), and AteThat API + Core (768 MB) | about 5.85 GB |
| Then reserve a shared local MongoDB (2 GB) | about 3.85 GB |
| Then reserve shared Redis (256 MB) | about 3.59 GB |
| Then reserve shared RabbitMQ (768 MB) | about 2.82 GB |

At idle, the host may appear to have roughly 6-7 GB free after AteThat starts because services do not immediately consume their full limits. That is not safe capacity to promise to other projects: the allocation table preserves headroom for traffic spikes, Go garbage collection, MongoDB caches, queue bursts, Linux file cache, and deployment overlap.

The recommended initial shared-platform layout keeps AteThat's MongoDB on Atlas M0 and does not run Redis or RabbitMQ. This leaves the approximately 5.85 GB planned headroom for other small projects. Add a local shared stateful service only when a project needs it, then deduct its budget from that headroom.

### 10. Future scaling and platform choices

KVM 2 is the deliberate initial choice. Do not commit to KVM 4 based on projected account count alone; select the next step from measured CPU, memory, latency, disk, and isolation needs.

| Need observed in production | Preferred next step |
|---|---|
| One shared VM needs more CPU, RAM, or disk capacity | Upgrade to KVM 4 |
| Stateful services or projects need a separate failure and security boundary | Add a second KVM 2 and place workloads deliberately across the two VMs |
| A project needs a queue or cache | Add Redis or RabbitMQ only for that concrete requirement, with an explicit resource limit |
| Atlas M0 becomes constrained | Move to an appropriate Atlas tier or deploy self-managed MongoDB, using the same Core Service persistence code |

Provider selection and future scaling should continue to prefer INR billing and avoid unnecessary foreign-exchange/card-conversion exposure. Any future self-managed MongoDB, Redis, or RabbitMQ service remains private to the VM network. For shared use across the owner's small projects, each service uses separate project credentials and namespace boundaries.

### 11. Initial monitoring, logging, and alerting

The following is a proposed lightweight monitoring model, not an inventory of verified running containers. Release checks verified API/Core health and rotated container logs; UptimeRobot was selected, but this document does not establish that every monitor/tool below is deployed. Recheck before allocating resources or claiming alert coverage.

Use a lightweight hybrid monitoring model. Self-hosted tools provide VM and container visibility; an external monitor detects a full KVM, Docker, network, or reverse-proxy outage that cannot be observed from the failed VM itself.

| Need | Initial tool | Deployment rule |
|---|---|---|
| External public uptime monitoring | UptimeRobot Free | Monitor each public project URL from outside the KVM at its available free-tier interval. |
| Internal and endpoint health monitoring | Uptime Kuma | Monitor public frontend/API URLs and private service health endpoints; send alerts through an approved notification channel. |
| VM and container resource history | Beszel | Track CPU, RAM, disk, network, and Docker container usage; configure alerts for sustained pressure. |
| Container log retention | Docker `json-file` logging with rotation | Configure `max-size: 10m` and `max-file: 3` as the host default unless a project needs a different explicit limit. |
| Live log inspection | Dozzle | Keep private behind authenticated administrative access; never expose its Docker-socket access publicly. |
| Scheduled-job failure monitoring | Healthchecks.io Free | Use when recurring jobs exist, including future data-compaction or maintenance jobs. |

The initial monitoring budget is approximately 256-512 MB of KVM memory for Uptime Kuma, Beszel, and occasional Dozzle use. UptimeRobot and Healthchecks.io are external services and use no KVM resources.

Do not initially deploy Prometheus, Grafana, Loki, Netdata, or self-hosted Sentry. Revisit centralized metrics and retained log search only when the small tooling set cannot answer operational questions; reserve roughly 0.5-1 GB RAM plus retained-log disk capacity when adopting a Grafana/Loki stack.

Application logs must be structured where practical and must not include passwords, tokens, full food details, health notes, body measurements, or complete provider responses.

## Phase 2 — completed deployment

Release [v2.0.0](https://github.com/architsangal/AteThat/releases/tag/v2.0.0) pins production commit `4e649a2a2ca1ba23395f98f415d71ed6d7a6bd03`. API/Core were healthy after [the production deployment](https://github.com/architsangal/AteThat/actions/runs/38879140661); authorized Chrome test-account sanity covered nutrition, recipes, wellness and schedules. No new service/server/provider subscription was required. See [release evidence](PHASE_2_STATUS.md).

Frontend production is the connected Worker `atethat`, built from `main`. Vite configuration is compiled into static assets; runtime bindings are not needed for the frontend. Its API base is `https://api.atethat.architsangal.in/api/v1`. Non-production builds must use `npx wrangler preview` with `previews: {}` in Wrangler; do not rename the connected Worker to `dev-atethat` or use the production deploy command for previews.

CI typechecks/builds web, runs all frontend regressions and Go tests, provisions a disposable MongoDB replica set for Core integration tests, and validates Compose. Phase 2.5 added hosted integration coverage; local HTTP and Chrome checks remain separate release evidence. The backend pipeline rebuilds on the VPS; registry/image-digest deployment remains future hardening. Historical checkout prices/resource budgets above are planning records, not newly verified prices or measured free capacity.

## Phase 3 — setup gates, not deployed infrastructure

The catalogue reuses Atlas with a separate logical food database and least-privilege Core access; no Redis or new VPS. Approve database names, connection limits, measured document/index budget, quota storage and migration before deploying. Add non-secret configuration and tests without exposing database/provider credentials to the browser.

AI/transcription may introduce external provider costs and health-data processing; select providers, consent/retention and spending limits before enabling them. Exports/agent grants require bounded storage/jobs and revocable access. No provider purchase or separate worker is authorized yet. Follow feature → `dev` → `main`, local tests and authorized production sanity for each milestone.

## Phase 4 and later — operational evolution

Broader workers/queues, metrics pipelines, notifications, offline sync and scaling are conditional [platform backlog](PHASE_4_BACKLOG.md). Backup/recovery remains excluded pending explicit approval. Code tags do not roll back MongoDB data, and deployment must not blindly reset data to an older schema.
