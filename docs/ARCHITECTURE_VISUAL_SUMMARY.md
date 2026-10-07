# AteThat — Architecture Visual Summary

| Field | Value |
|---|---|
| Purpose | Visual companion to the system, database, and deployment designs |
| Scope | Phase 1 architecture and the agreed deployment direction |
| Source of truth | `3. SYSTEM_DESIGN.md`, `4. DATABASE_DESIGN.md`, and `DEPLOYMENT.md` |

This document shows the important boundaries, request flows, data passed between layers, and database storage model. It summarizes rather than replaces the source design documents.

## 1. Complete system and deployment view

```mermaid
flowchart LR
    User[User browser]
    DNS[DNS<br/>architsangal.in]
    Pages[Cloudflare Pages<br/>React frontend static files]
    Edge[Caddy on Hostinger KVM<br/>public ports 80 and 443]
    API[Go API Service<br/>REST /api/v1]
    Core[Go Core Service<br/>private gRPC]
    DB[(MongoDB Atlas M0<br/>initial database)]
    USDA[USDA FoodData Central]
    SMTP[Gmail SMTP]
    Google[Google OAuth / OIDC]

    User -->|HTTPS frontend hostname| DNS --> Pages
    User -->|HTTPS API hostname| DNS --> Edge
    Edge -->|API hostname| API
    API -->|authenticated gRPC request| Core
    Core -->|TLS database connection| DB
    Core -->|HTTPS| USDA
    Core -->|SMTP TLS| SMTP
    Core -->|OAuth/OIDC TLS| Google

    classDef public fill:#dbeafe,stroke:#2563eb,color:#1e3a8a
    classDef private fill:#dcfce7,stroke:#16a34a,color:#14532d
    classDef external fill:#fef3c7,stroke:#d97706,color:#78350f
    class User,DNS,Pages,Edge,API public
    class Core private
    class DB,USDA,SMTP,Google external
```

### Public and private boundaries

| Location | Accessible from the internet? | What it handles |
|---|---|---|
| Cloudflare Pages | Yes | TLS static frontend delivery |
| Caddy | Yes | API TLS, HTTP-to-HTTPS redirect, hostname routing |
| API Service | Yes, through Caddy | REST input validation, rate limits, authentication, HTTP-to-gRPC translation |
| Core Service | No | Domain rules, persistence, USDA access, email sending |
| MongoDB | No browser/API access | Authoritative user data |

The browser never receives database credentials, USDA credentials, Gmail credentials, or Core Service access.

## 2. What crosses each application boundary

```mermaid
flowchart TB
    FE[Frontend request DTO<br/>user-entered fields only]
    API[API Service<br/>validates REST shape and access token]
    GRPC[gRPC request<br/>validated fields + authenticated principal]
    CORE[Core Service<br/>business rules and ownership checks]
    STORE[MongoDB documents<br/>server-generated IDs, hashes, snapshots, timestamps]

    FE -->|HTTPS REST| API -->|private gRPC| GRPC --> CORE -->|MongoDB driver| STORE

    classDef client fill:#dbeafe,stroke:#2563eb,color:#1e3a8a
    classDef service fill:#dcfce7,stroke:#16a34a,color:#14532d
    classDef data fill:#f3e8ff,stroke:#9333ea,color:#581c87
    class FE,API,GRPC client
    class CORE service
    class STORE data
```

| Flow | Browser sends | Internal service receives/creates | Stored data |
|---|---|---|---|
| Registration | Email, password, agreed profile inputs | Normalized email; password is immediately hashed; verification token is created | User, auth identity, hashed verification token |
| Login | Email/password or Google OAuth start/callback | Authenticated user/session result; verified Google subject for Google sign-in | Password hash where applicable; Google provider/subject only; refresh-token hash and session metadata; never raw password, provider token, or refresh token |
| Food search | Search text, filters, cursor/page size | Authenticated principal plus normalized query | Short-lived shared USDA cache only when fetched |
| Food preview | Selected food IDs, serving/quantity, food state, logged timestamp | Canonical nutrition calculation and preview | Short-lived preview with expiry |
| Food confirmation | `preview_id`, idempotency key, final user confirmation | User ID from access token; transaction request | One group and one-or-more immutable food-log snapshots |
| Dashboard/history | Date range, cursor/page size | User ID from access token; Tracking Day calculations | Reads only; no client-supplied ownership is trusted |
| Health action | Item/schedule/event input and timestamp | User ID, schedule interpretation, UTC normalization | Health item, schedule version, or occurrence event |

## 3. Authentication and account recovery

```mermaid
sequenceDiagram
    actor U as User browser
    participant A as API Service
    participant C as Core Service
    participant D as MongoDB
    participant M as Gmail SMTP

    U->>A: POST /auth/register<br/>email, password
    A->>C: Register(validated email, password)
    C->>D: Store user + Argon2id password hash<br/>Store hashed verification token
    C->>M: Send verification link/token
    M-->>U: Verification email
    U->>A: Verify token, POST /auth/login,<br/>or backend-managed Google OAuth callback
    A->>C: Verify/Login/OIDC request
    C->>D: Validate user/token or password hash<br/>Store refresh-token hash and session family
    C-->>A: Access token + raw refresh token
    A-->>U: Session response
```

```text
Never stored or logged: raw password, raw refresh token, raw email-verification token,
database URI, Gmail app password, USDA key, or token-signing key.
```

## 4. Food logging: preview to immutable history

```mermaid
sequenceDiagram
    actor U as User browser
    participant A as API Service
    participant C as Core Service
    participant F as USDA / provider cache
    participant D as MongoDB

    U->>A: Search query, filters, cursor
    A->>C: SearchFoods(user principal, normalized query)
    C->>F: Read unexpired cache or call USDA
    F-->>C: Canonical food data
    C-->>A: Paginated food results
    A-->>U: Foods, servings, nutrient availability

    U->>A: Create preview<br/>multiple foods, quantities, logged_at
    A->>C: PreviewFoodLog(user principal, selections)
    C-->>A: Preview ID, item totals, aggregate totals, expiry
    A-->>U: Reviewable nutrition preview

    U->>A: Confirm preview<br/>preview_id, idempotency_key
    A->>C: ConfirmFoodLogGroup(user principal, request)
    C->>D: One transaction:<br/>group + all item snapshots
    D-->>C: Confirmed IDs and timestamps
    C-->>A: Confirmed group
    A-->>U: Final saved result
```

The confirmation transaction is all-or-nothing. A group may contain many food items; it never creates a partially confirmed meal.

## 5. Food-log data shape and ownership

```mermaid
flowchart TB
    User[(users)]
    Preview[food_log_previews<br/>short-lived, not history]
    Group[food_log_groups<br/>one confirmed meal/log action]
    Item[food_logs<br/>one immutable snapshot per food item]
    Target[nutrition_target_versions]
    Cache[provider_cache<br/>USDA cache with TTL]

    User -->|user_id| Preview
    User -->|user_id| Group
    User -->|user_id| Item
    User -->|user_id| Target
    Preview -. confirmation input .-> Group
    Group -->|food_log_group_id| Item
    Cache -. normalizes food data for .-> Preview
    Target -. dashboard comparison .-> Group

    classDef durable fill:#f3e8ff,stroke:#9333ea,color:#581c87
    classDef temporary fill:#fef3c7,stroke:#d97706,color:#78350f
    class User,Group,Item,Target durable
    class Preview,Cache temporary
```

| Collection | Main fields | Why it exists |
|---|---|---|
| `food_log_groups` | `user_id`, Tracking Day, logged UTC time, status, item count, aggregate final nutrition, revision | Represents one user confirmation action, including a multi-food meal |
| `food_logs` | `user_id`, group ID, food source/name snapshot, quantity, provider/final nutrition snapshots, timestamps | Preserves each food item as historical evidence |
| `food_log_previews` | User ID, chosen items, calculated totals, expiry | Lets the user review totals before persistence |
| `provider_cache` | USDA cache key, normalized food/search payload, fetched/expiry timestamps | Reduces repeated provider calls; never becomes permanent history |

All user-owned collections carry `user_id`, and Core Service queries bind it from the authenticated session—not from a request-body field.

## 6. Other stored user data

```mermaid
erDiagram
    USERS ||--|| USER_PROFILES : owns
    USERS ||--|| USER_PREFERENCES : owns
    USERS ||--o{ AUTH_IDENTITIES : authenticates_with
    USERS ||--o{ SESSIONS : has
    USERS ||--o{ EMAIL_TOKENS : receives
    USERS ||--o{ NUTRITION_TARGET_VERSIONS : sets
    USERS ||--o{ CUSTOM_FOODS : creates
    USERS ||--o{ MEALS : saves
    USERS ||--o{ FOOD_LOG_GROUPS : confirms
    FOOD_LOG_GROUPS ||--|{ FOOD_LOGS : contains
    USERS ||--o{ HEALTH_ITEMS : tracks
    HEALTH_ITEMS ||--o{ HEALTH_ITEM_SCHEDULE_VERSIONS : versions
    HEALTH_ITEMS ||--o{ HEALTH_ITEM_EVENTS : records
```

| Data family | Storage rule |
|---|---|
| Identity and sessions | Passwords/tokens are stored only as hashes; expired session/token documents use TTL indexes |
| Profile and preferences | Stores user default timezone and Tracking Day preferences; timestamps remain UTC |
| Targets | Versions are retained so historical comparisons remain explainable |
| Custom foods and meals | Reusable user-owned definitions; changing them does not rewrite confirmed food history |
| Health items | Definitions, schedule versions, and recorded events remain distinct from food logs |

## 7. Detailed-history compaction

> This is an agreed Phase 1 retention rule and is represented in the database design.

```mermaid
flowchart LR
    Raw[Detailed food logs<br/>0 to 90 days<br/>food names, servings, item snapshots]
    Daily[Daily nutrition summary<br/>one user + Tracking Day<br/>final totals and counts]
    Monthly[7-day and monthly views<br/>computed from daily summaries]
    Delete[Delete detailed item/group history<br/>only after daily summary is verified]

    Raw -->|scheduled compaction after 90 days| Daily
    Daily --> Monthly
    Daily --> Delete

    classDef raw fill:#dbeafe,stroke:#2563eb,color:#1e3a8a
    classDef durable fill:#dcfce7,stroke:#16a34a,color:#14532d
    class Raw raw
    class Daily,Monthly,Delete durable
```

```text
Before 90 days: exact foods and individual servings are available.
After 90 days: daily totals remain available, but exact food-item history is no longer available.
```

Daily summaries are sufficient for weekly and monthly calculations; a permanent monthly copy is not required unless later performance measurements justify a materialized rollup. Compacted days are read-only because their item-level evidence no longer exists.

## 8. Database deployment portability

```mermaid
flowchart LR
    Core[Core Service<br/>one MongoDB driver and repository implementation]
    Config[Protected deployment configuration<br/>MongoDB URI + database name]
    Atlas[(MongoDB Atlas M0<br/>initial)]
    Local[(MongoDB Community Docker container<br/>future option)]

    Core --> Config
    Config -->|Atlas URI| Atlas
    Config -->|private Docker URI| Local

    classDef app fill:#dcfce7,stroke:#16a34a,color:#14532d
    classDef db fill:#f3e8ff,stroke:#9333ea,color:#581c87
    class Core,Config app
    class Atlas,Local db
```

Both database targets use the same collections, indexes, and transaction behavior. If local MongoDB is adopted, it must be authenticated, private to Docker networking, persist to a volume, and run as a single-node replica set.

## 9. Monitoring, logging, and alerting view

```mermaid
flowchart LR
    Internet[External internet]
    UR[UptimeRobot Free<br/>external uptime check]
    KVM[Hostinger KVM]
    Caddy[Caddy / public URLs]
    Kuma[Uptime Kuma<br/>endpoint and internal health]
    Beszel[Beszel<br/>host and container metrics]
    Docker[Docker JSON logs<br/>rotated]
    Dozzle[Dozzle<br/>private live-log viewer]
    HC[Healthchecks.io<br/>scheduled-job heartbeats]
    Alert[Approved notification channel]

    Internet --> UR --> Caddy
    KVM --> Caddy
    Kuma --> Caddy
    Kuma --> Alert
    Beszel --> Alert
    Docker --> Dozzle
    HC --> Alert

    classDef external fill:#fef3c7,stroke:#d97706,color:#78350f
    classDef local fill:#dcfce7,stroke:#16a34a,color:#14532d
    classDef private fill:#f3e8ff,stroke:#9333ea,color:#581c87
    class Internet,UR,HC,Alert external
    class KVM,Caddy,Kuma,Beszel,Docker local
    class Dozzle private
```

An external uptime monitor is essential because KVM-local monitoring cannot report an outage when the entire VM, Docker daemon, or network path is unavailable.

## 10. Capacity and future scaling choices

```mermaid
flowchart TD
    Start[KVM 2<br/>2 vCPU, 8 GB RAM]
    Measure[Measure CPU, RAM, disk, latency,<br/>error rate, and isolation needs]
    NeedCapacity{One shared VM<br/>needs more capacity?}
    NeedIsolation{Need separate failure/security<br/>boundary?}
    KVM4[Upgrade to KVM 4]
    KVM2B[Add a second KVM 2]
    Optional[Add local stateful service only when required<br/>MongoDB, Redis, or RabbitMQ]

    Start --> Measure --> NeedCapacity
    NeedCapacity -- Yes --> KVM4
    NeedCapacity -- No --> NeedIsolation
    NeedIsolation -- Yes --> KVM2B
    NeedIsolation -- No --> Optional
    KVM4 --> Measure
    KVM2B --> Measure
    Optional --> Measure
```

KVM 2 is the current cost-first deployment choice. Redis and RabbitMQ are optional future components, not initial AteThat dependencies.
