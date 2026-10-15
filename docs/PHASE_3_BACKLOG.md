# Phase 3 backlog: local-first USDA food catalogue

Recorded: 2026-10-16. The owner explicitly moved this proposal out of Phase 2. This document records scope and pending design decisions; it does not authorize implementation, paid services, pushing, or deployment.

## Intended behaviour

- Reuse the existing MongoDB deployment with a separate logical food database (proposed name: `atethat_foods`). No Redis or new server for this feature.
- Default search uses the shared locally stored USDA catalogue and the current user's private custom foods, without an implicit USDA search request.
- Provide an explicit "Search USDA" action for discovery, limited to 10 searches per authenticated user per day, not 10 for the whole application.
- Import a bounded page of successful discovery results, deduplicated by USDA food ID, so future local searches can find them. Preserve distinct USDA IDs even when names match. Load full nutrition when a user opens/selects an item; incomplete records must not be treated as complete nutrition.
- Store only necessary search, nutrition, serving, provenance, and normalization-version fields. Do not import images, unbounded search pages, or entire raw provider payloads.
- Keep catalogue records until explicit manual cleanup; do not automatically delete after 90 days of inactivity. Track fetch time, last meaningful use, and usage count. Merely appearing in search does not count as meaningful use.
- Keep source records shared/read-only to ordinary users. One-off overrides affect only the log; reusable edits become private custom foods. Confirmed nutrition snapshots remain independent of catalogue refresh/deletion.
- Enforce daily quotas atomically on the backend. Successful empty searches count; provider failures do not. Food details/refresh calls have separate upstream accounting, not the user's search allowance.
- Add an application-wide upstream budget, including search/detail/refresh requests, respecting the actual USDA response-header limits. During provider cooldown/failure, show "USDA search is temporarily unavailable. Please try again later." Recover automatically; local search/custom foods remain usable.
- Initial proposed catalogue storage budget: 20 MB including indexes, to be validated against actual MongoDB document/index measurements. Earlier food-count estimates were illustrative, not guaranteed capacity. Stop growth safely at the budget and expose an operational reason; upstream recovery cannot solve local storage exhaustion.
- Cleanup is manual and must not delete diary history. A future cleanup tool should offer a dry run and protect active food references by defining safe behaviour for recipes/meals/servings that refer to removed catalogue IDs.
- Monitoring dashboards/alerts are secondary follow-on work, not a prerequisite to scope discussion. Basic error handling and quota enforcement remain part of the feature itself.

## Decisions to resolve before implementation

- Daily quota boundary: UTC midnight was proposed but not yet approved; display the resulting reset time clearly.
- Nutrition freshness: a 30-day stale interval with refresh on use was proposed, not approved. Retention is indefinite until cleanup; freshness is a separate policy. Decide stale-data fallback and avoid unnecessary provider calls during logging.
- Storage budget measurement/enforcement and behaviour when importing a result would exceed it; catalogue metadata and quota collections also consume deployment capacity.
- Search indexing/ranking, per-result import completeness, pagination costs, quota treatment of reused discovery responses, and request deduplication.
- Safe migration from the current expiring USDA cache. This proposal supersedes the current maximum-15-day provider-cache policy only when implemented and the architecture documents are updated; it does not change current runtime behaviour.

## Verification and release

Test locally before authorized feature-to-dev and dev-to-main promotion. Cover ownership isolation, duplicate IDs, distinct same-name foods, quota concurrency/reset, provider failures/cooldown, storage exhaustion, refresh failure, manual cleanup references, and immutable historical nutrition.

Barcode scanning is removed from scope and is not part of this backlog. Expanded Indian-food providers require separate evaluation/approval; no paid subscription is assumed.
