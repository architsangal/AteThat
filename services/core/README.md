# Core application service

Private gRPC Go service owning domain rules, MongoDB, SMTP and USDA normalization. Never expose Core directly to the browser or give external agents database credentials.

## Phase 1 — released

Identity/profile/preferences, Tracking Day, FDA general benchmarks, USDA/cache, private foods/meals/servings, decimal preview/idempotent confirmation, immutable history and basic health scheduling are implemented. Account deletion includes cancellation grace and hourly purge.

The actual `usda_food_cache` contains raw response bytes with 24-hour search/detail expiry; it is not the planned persistent catalogue.

## Phase 2 — released

Recipes/yields/portions/overrides, expanded nutrition analysis, Decimal128 wellness records/settings and versioned monthly/multi-time/pause schedules extend existing ownership and snapshot rules. Safe nutrient identifiers include B6/B12. See [implementation](../../docs/PHASE_2_IMPLEMENTATION.md).

## Phase 3 — planned

Separate logical MongoDB food catalogue, local-first search, atomic 10-search/user/day discovery, global upstream budget, bounded normalized imports and manual cleanup. AI proposals/voice, exports and scoped agent access reuse domain validation; minimum agent audit/revocation are required. Provider/privacy/cost decisions remain open. See [backlog](../../docs/PHASE_3_BACKLOG.md).

## Phase 4 and later — deferred

Broader outbox/worker/audit/analytics, notifications and scaling are conditional. No Redis/broker or recovery service is approved.

## Verification

Run `go test ./...`; Mongo-backed integration needs `ATETHAT_TEST_MONGO_URI` for an isolated replica set. Linux containers avoid the observed native macOS Go toolchain issue. See [local setup](../../deploy/local/README.md). Never use production secrets/data for local tests.
