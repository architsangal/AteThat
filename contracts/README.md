# AteThat contracts

Browser → REST API → private gRPC Core. Contracts may be consumed by both services; generated code belongs in each consumer module, not a shared runtime helper package. See [API design](../docs/5.%20API_DESIGN.md).

## Phase 1 — released contracts

Private protobuf contracts cover identity/profile/targets, foods/meals/logging, dashboard/history and health domains. The public OpenAPI document currently formalizes a subset (27 operations), including identity/account flows; it is not the complete runtime route inventory. Account deletion is implemented with a 30-day grace period, not an unresolved design choice.

## Phase 2 — released additions

Protobuf and runtime REST add recipe portion/override fields, wellness services/settings and advanced schedules/history. Markdown API design records the additive shapes. Full OpenAPI route formalization remains documentation debt; do not treat a successful subset check as complete coverage.

## Phase 3 — proposed contracts

Catalogue/discovery/quota metadata, assisted proposals, exports and agent scopes/tools need reviewed contracts before implementation. Final paths/tool names and quota reset/error semantics are not frozen. Keep decimal strings, owner enforcement, unknown nutrition and explicit confirmation consistent across clients. See [backlog](../docs/PHASE_3_BACKLOG.md).

## Phase 4 and later — conditional contracts

Notification/offline/job and broader audit APIs remain deferred. Agent minimum authorization/audit must accompany Phase 3 agents.

## Generation and checks

Run `make contracts-check` to parse OpenAPI/references/operation IDs and compile protobuf descriptors. Use the repository contract-generation targets/scripts when changing protobuf and review generated consumer changes together. Do not bypass typed contracts with untyped JSON envelopes.
