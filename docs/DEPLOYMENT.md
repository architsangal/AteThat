# AteThat - Deployment

| Field | Value |
|---|---|
| Status | Planning; no infrastructure has been purchased or deployed |
| Scope | Hosting, domain, routing, TLS, and deployment-cost decisions |
| Product | AteThat |

## 1. Current deployment decision

The initial hosting candidate is **Hostinger KVM 2** in the India region.

| Resource | Initial choice |
|---|---|
| VM | Hostinger KVM 2 |
| VM resources | 2 vCPU, 8 GB RAM, 100 GB NVMe storage, 8 TB transfer |
| Region | India; do not use Malaysia for the India-first product unless the final latency test changes this decision |
| Database | MongoDB Atlas Free cluster (`M0`) initially |
| Database recovery | No application-managed backups or recovery guarantee in the initial configuration |
| Reverse proxy | Caddy |
| Containers | Docker Compose, with one isolated Compose project and private network per application |
| TLS | Caddy-managed public TLS certificates and automatic HTTP-to-HTTPS redirects |
| Frontend | Cloudflare Pages serves the AteThat static frontend |
| DNS | Frontend hostname points to Cloudflare Pages; API hostname points to the VM |

KVM 2 is the selected cost-first starting point for AteThat and several small, low-traffic projects. Increase capacity only after resource monitoring and load testing show that the VM is constrained. KVM 4 remains the vertical-scale option; it is not the initial choice.

## 2. Cost record

The Hostinger 24-month KVM 2 checkout price observed during planning was:

| Item | Amount |
|---|---:|
| Base price for 24 months | ₹19,176.00 |
| GST | ₹3,451.68 |
| Total payable | ₹22,627.68 |
| Published renewal price | ₹1,199/month before GST for a 24-month renewal |

Final checkout pricing, taxes, and promotion eligibility must be checked before purchase.

## 3. Coupon record

| Field | Value |
|---|---|
| Code | `GXDGLOBAL` |
| Claimed discount | 20% off |
| Intended use | Hostinger checkout for the initial VPS purchase |
| Verification status | User supplied; apply it at checkout and confirm that the final total is lower before paying |

Use one promotion only. Do not assume a coupon changes the published renewal price unless Hostinger's checkout explicitly says so.

## 4. Domain and routing model

The planned parent domain is `architsangal.in`. Each project has separate public hostnames so projects do not overlap:

```text
atethat.architsangal.in       -> Cloudflare Pages (AteThat frontend)
api.atethat.architsangal.in   -> Hostinger KVM / Caddy / AteThat API Service

rollercoaster.architsangal.in     -> RollerCoaster frontend (future)
api.rollercoaster.architsangal.in -> RollerCoaster API (future)
```

The domain is deployment configuration, not application code. Caddy is the only service listening publicly on ports 80 and 443. It routes each recognized hostname only to its matching project; unknown hostnames are rejected.

## 5. Initial VM layout

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

## 6. Pre-deployment checklist

- [ ] Purchase or renew the parent domain and keep registrar access secure.
- [ ] Confirm the Hostinger checkout final total after applying `GXDGLOBAL`.
- [ ] Select the India VM region.
- [ ] Install a supported Linux LTS image and apply security updates.
- [ ] Configure firewall rules: publicly allow only TCP 80 and 443; restrict administrative access separately.
- [ ] Create DNS records for the AteThat frontend and API hostnames.
- [ ] Connect `atethat.architsangal.in` to Cloudflare Pages and configure the public API base URL.
- [ ] Configure Caddy with persistent certificate storage and hostname-specific routes.
- [ ] Configure Atlas IP access to allow only the VM's outbound address.
- [ ] Store production secrets outside Git and supply them through protected environment files or a secret manager.
- [ ] Deploy the container images with a unique Compose project name and private network.
- [ ] Verify HTTPS, HTTP-to-HTTPS redirect, login, email delivery, food logging, and health checks.
- [ ] Record baseline CPU, memory, disk, and response-time measurements before adding further projects.

## 7. Scaling triggers

Keep KVM 2 while it has sufficient measured headroom. Reassess the VM tier when sustained resource use, not account count alone, indicates pressure:

- Sustained memory usage reaches approximately 70-75% after normal operating headroom.
- Sustained CPU saturation, increased request latency, or swapping occurs under a representative load test.
- Additional projects require persistent databases, background workers, build workloads, or materially higher traffic.
- MongoDB Atlas M0 becomes a data-size, throughput, or connection bottleneck.

The architecture uses containers and hostname-based routing so moving from KVM 2 to KVM 4 is a deployment resize/migration, not an application rewrite.

## 8. KVM 2 container resource budget

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

### 8.1 Practical KVM 2 scenarios

| Deployment shape | Approximate reserved RAM | KVM 2 assessment |
|---|---:|---|
| Caddy + AteThat services + Atlas M0 | 2.1 GB | Comfortable starting point with room for several small projects. |
| Above + local MongoDB | 4.1 GB | Viable, but reduces room for other projects; monitor closely. |
| Above + local MongoDB + Redis + RabbitMQ | 5.1 GB | Technically viable for AteThat and one or two small projects, but not the preferred layout for five or six projects. |

For AteThat Phase 1, do not add Redis or RabbitMQ unless a concrete requirement needs them. The current design uses MongoDB TTL-backed provider caching and synchronous gRPC between the API and Core services, so neither dependency is needed merely because containers are being deployed.

If multiple projects eventually require their own databases, queues, and caches on the same VM, that is a valid trigger to move from KVM 2 to KVM 4. Database, Redis, and RabbitMQ ports remain private; Caddy exposes only HTTP/HTTPS.

### 8.2 Shared-platform allocation after AteThat

Only the AteThat API and Core Service are AteThat-specific. Caddy can be shared by every project. MongoDB, Redis, and RabbitMQ can technically be shared platform services, but every project must have separate credentials and namespace boundaries (MongoDB database/user, Redis ACL/key prefix, RabbitMQ virtual host/user). Sharing them also creates a shared failure domain, so do this only for small, low-risk projects.

| Allocation on KVM 2 | RAM remaining from 8 GB |
|---|---:|
| After reserving Linux/Docker (1.25 GB), shared Caddy (128 MB), and AteThat API + Core (768 MB) | about 5.85 GB |
| Then reserve a shared local MongoDB (2 GB) | about 3.85 GB |
| Then reserve shared Redis (256 MB) | about 3.59 GB |
| Then reserve shared RabbitMQ (768 MB) | about 2.82 GB |

At idle, the host may appear to have roughly 6-7 GB free after AteThat starts because services do not immediately consume their full limits. That is not safe capacity to promise to other projects: the allocation table preserves headroom for traffic spikes, Go garbage collection, MongoDB caches, queue bursts, Linux file cache, and deployment overlap.

The recommended initial shared-platform layout keeps AteThat's MongoDB on Atlas M0 and does not run Redis or RabbitMQ. This leaves the approximately 5.85 GB planned headroom for other small projects. Add a local shared stateful service only when a project needs it, then deduct its budget from that headroom.

## 9. Future scaling and platform choices

KVM 2 is the deliberate initial choice. Do not commit to KVM 4 based on projected account count alone; select the next step from measured CPU, memory, latency, disk, and isolation needs.

| Need observed in production | Preferred next step |
|---|---|
| One shared VM needs more CPU, RAM, or disk capacity | Upgrade to KVM 4 |
| Stateful services or projects need a separate failure and security boundary | Add a second KVM 2 and place workloads deliberately across the two VMs |
| A project needs a queue or cache | Add Redis or RabbitMQ only for that concrete requirement, with an explicit resource limit |
| Atlas M0 becomes constrained | Move to an appropriate Atlas tier or deploy self-managed MongoDB, using the same Core Service persistence code |

Provider selection and future scaling should continue to prefer INR billing and avoid unnecessary foreign-exchange/card-conversion exposure. Any future self-managed MongoDB, Redis, or RabbitMQ service remains private to the VM network. For shared use across the owner's small projects, each service uses separate project credentials and namespace boundaries.

## 10. Initial monitoring, logging, and alerting

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
