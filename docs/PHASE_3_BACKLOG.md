# Phase 3 backlog — catalogue, assisted input and integrations

Recorded: 2026-10-16. The owner explicitly moved this proposal out of Phase 2. This document records scope and pending design decisions; it does not authorize implementation, paid services, pushing, or deployment.

## Phase 1 — completed foundation

Release `v1.0.0` established ownership, Core-only persistence, attributed nutrition, immutable reviewed logging and the original provider cache.

## Phase 2 — completed foundation

Release `v2.0.0` added recipes, richer analysis, wellness and advanced schedules. AI/voice/agents were excluded from Phase 2, not cancelled from the original Phase 3 roadmap. Barcode was removed entirely.

## Phase 3 — planned work

### A. Agreed addition: local-first food catalogue

The following captures the owner's agreed direction. Specific daily boundary, freshness, storage accounting and edge-case policies remain proposals until resolved.

### Catalogue behaviour

- Reuse the existing MongoDB deployment with a separate logical food database (proposed name: `atethat_foods`). No Redis or new server for this feature.
- Default search uses the shared locally stored USDA catalogue and the current user's private custom foods, without an implicit USDA search request.
- Provide an explicit "Search USDA" action for discovery, limited to 10 searches per authenticated user per day, not 10 for the whole application.
- Import a bounded page of successful discovery results, deduplicated by USDA food ID, so future local searches can find them. Preserve distinct USDA IDs even when names match. Load full nutrition when a user opens/selects an item; incomplete records must not be treated as complete nutrition.
- Store only necessary search, nutrition, serving, provenance, and normalization-version fields. Do not import images, unbounded search pages, or entire raw provider payloads.
- Keep catalogue records until explicit manual cleanup; do not automatically delete after 90 days of inactivity. Track fetch time, last meaningful use, and usage count. Merely appearing in search does not count as meaningful use.
- Keep source records shared/read-only to ordinary users. One-off overrides affect only the log; reusable edits become private custom foods. Confirmed nutrition snapshots remain independent of catalogue refresh/deletion.
- Enforce daily quotas atomically on the backend. Proposed accounting: successful empty searches count; provider failures do not. Resolve reservation/refund rules before implementation; an upstream failed request may still consume the provider's own quota. Food details/refresh calls have separate upstream accounting, not the user's search allowance.
- Add an application-wide upstream budget, including search/detail/refresh requests, respecting the actual USDA response-header limits. During provider cooldown/failure, show "USDA search is temporarily unavailable. Please try again later." Recover automatically; local search/custom foods remain usable.
- Initial proposed catalogue storage budget: 20 MB including indexes, to be validated against actual MongoDB document/index measurements. Earlier food-count estimates were illustrative, not guaranteed capacity. Stop growth safely at the budget and expose an operational reason; upstream recovery cannot solve local storage exhaustion.
- Cleanup is manual and must not delete diary history. A future cleanup tool should offer a dry run and protect active food references by defining safe behaviour for recipes/meals/servings that refer to removed catalogue IDs.
- Monitoring dashboards/alerts are secondary follow-on work, not a prerequisite to scope discussion. Basic error handling and quota enforcement remain part of the feature itself.

### Catalogue decisions to resolve before implementation

- Daily quota boundary: UTC midnight was proposed but not yet approved; display the resulting reset time clearly.
- Nutrition freshness: a 30-day stale interval with refresh on use was proposed, not approved. Retention is indefinite until cleanup; freshness is a separate policy. Decide stale-data fallback and avoid unnecessary provider calls during logging.
- Storage budget measurement/enforcement and behaviour when importing a result would exceed it; catalogue metadata and quota collections also consume deployment capacity.
- Search indexing/ranking, per-result import completeness, pagination costs, quota treatment of reused discovery responses, and request deduplication.
- Safe migration from the current expiring USDA cache. The current implementation uses a 24-hour raw-response cache; older design text proposed a maximum of 15 days. Neither is the persistent catalogue. Introduce the new retention policy only with the feature and an explicit migration; current runtime behaviour is unchanged.

### Catalogue verification and release

Test locally before authorized feature-to-dev and dev-to-main promotion. Cover ownership isolation, duplicate IDs, distinct same-name foods, quota concurrency/reset, provider failures/cooldown, storage exhaustion, refresh failure, manual cleanup references, and immutable historical nutrition.

Barcode scanning is removed from scope and is not part of this backlog. Expanded Indian-food providers require separate evaluation/approval; no paid subscription is assumed.

### B. Original roadmap: AI-assisted input

The historical PRD (Phase 3, pages 47–60) planned these capabilities; they remain backlog, not deployed features or permission to spend:

- Nutrition-label image extraction into editable nutrient/unit/serving-basis fields.
- Food-photo recognition into candidate foods, quantities and states; estimates/uncertainty must be visible.
- Natural-language logging resolved through normal food lookup and user serving definitions; ask for missing quantities rather than silently guess.
- Voice transcription feeding the same text/structured-intent pipeline.

All outputs are proposals. Core validates them and the user reviews/confirms food logging through the existing snapshot pipeline. AI never writes directly to MongoDB, invents successful logs, prescribes diets or changes medication dosage. Provider evaluation, cost limits, consent, upload size/retention/deletion and handling of health data must be approved before integration.

### C. Original roadmap: CSV/JSON exports

Provide one ownership-enforced Core export capability for the UI and authorized agents. Cover food logs/nutrient snapshots, meals/recipes, accepted targets, wellness, health items/actions and relevant profile fields. Export available daily summaries for compacted history; do not promise recovery of deleted item-level detail.

Design bounded ranges, pagination/streaming versus asynchronous jobs, decimal/time/source representations, download authentication and expiry. Exclude credentials/tokens. Test cross-user denial, large exports, partial failures and compacted ranges.

### D. Original roadmap: external agents and MCP

An MCP adapter uses existing application services, never database credentials. Candidate tools include food search/detail/reviewed logging, nutrition/history, health schedules/actions, goals/profile and exports; final names/contracts need design.

Require explicit user grants, least-privilege read/write/export scopes, revocation, normal ownership/version/idempotency rules and auditable confirmation for writes. Minimum audit records identify actor/agent/user, action/resource, time, outcome and request ID without copying secrets or unnecessary health payloads. This audit is a Phase 3 acceptance condition, even though the broader event-processing platform is Phase 4.

### Suggested milestones — proposal, not yet approved

1. Local-first catalogue and quotas.
2. CSV/JSON exports.
3. Label and natural-language assistance, then voice/food-photo input.
4. Scoped external-agent/MCP tools and their required audit/security tests.

For each milestone: define acceptance/cost/privacy limits → implement/test locally → feature PR to `dev` → release PR to `main` → authorized deployment and browser sanity. No date or paid provider is committed.

## Phase 4 and later — boundary

Broader worker/outbox infrastructure, platform-wide audit/analytics, notifications and offline/PWA remain in [Phase 4](PHASE_4_BACKLOG.md). Phase 3 may need bounded jobs, but not an automatically expanded service fleet. Monitoring dashboards are follow-on; basic logs, provider budgets and error handling are part of catalogue correctness. Regional dataset expansion and recovery need separate approval; barcode is not returning.
