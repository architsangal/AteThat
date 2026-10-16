# Public API service

Independently buildable Go module owning public `/api/v1` REST transport, bearer validation, CORS, cookies, limits and private gRPC translation. Never import Core internals or access MongoDB/USDA/SMTP directly.

## Phase 1 — released

Authentication, Google sign-in/explicit linking, profile/preferences, targets, foods/logging/meals, health schedules/history and verified account deletion with 30-day cancellation grace are implemented.

## Phase 2 — released

Recipe metadata/portion previews, expanded nutrition, wellness CRUD/settings and advanced schedule history are exposed through the same ownership boundary. Paused null-time and B6/B12 regressions are fixed. See [API design](../../docs/5.%20API_DESIGN.md).

## Phase 3 — planned

Catalogue/discovery allowance, assisted proposals, owned exports and scoped agent/MCP operations need explicit REST/gRPC contracts and failure/permission tests. No route is implemented merely by listing it in the [backlog](../../docs/PHASE_3_BACKLOG.md).

## Phase 4 and later — deferred

Notification/offline/job administration and wider audit integration are conditional platform work. Agent minimum audit is required with Phase 3 agents.

## Verification

Run `go test ./...` here and `make contracts-check` at the root. OpenAPI currently covers a subset, not all runtime routes. See [contracts](../../contracts/README.md).
