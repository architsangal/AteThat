# AteThat documentation and phase roadmap

## Phase 2.5 — local work, policy-gated

See [engineering requirements](PHASE_2_5_ENGINEERING_REQUIREMENTS.md) and [implementation status](PHASE_2_5_STATUS.md). Independent Settings targets, owner-approved assumed inputs, explanation page and factual Text Summary are implemented locally. Activity-based calculated defaults and previewed reset are now implemented locally; this does not change the Phase 2 production release status. Age/sex are not invented, and water is excluded.

Reconciled: 2026-10-16. Owner scope decisions override historical proposals. Release tags are code checkpoints, not database backups or a recovery guarantee.

## Phase 1 — complete

[v1.0.0](https://github.com/architsangal/AteThat/releases/tag/v1.0.0), commit `a5fa61f4afc403b4b8c38351fb34087e206596a0`: authentication, profile, general FDA benchmarks, food search/custom foods, reviewed logging, meals, basic schedules, dashboard/history and production infrastructure.

## Phase 2 — complete

[v2.0.0](https://github.com/architsangal/AteThat/releases/tag/v2.0.0), commit `4e649a2a2ca1ba23395f98f415d71ed6d7a6bd03`: expanded nutrition analysis, recipes, water/weight/activity, advanced schedules and supporting UX. See [implementation](PHASE_2_IMPLEMENTATION.md) and [verification/status](PHASE_2_STATUS.md).

Barcode scanning was removed, not postponed. Additional Indian-food providers were not selected. Notifications, offline/PWA, AI and backup/recovery implementation were excluded from Phase 2. These decisions supersede the original PDF Phase 2 acceptance list.

## Phase 2.1 — local UX verification

[Everyday UX improvements and audit traceability](PHASE_2_1_STATUS.md) are implemented and tested locally. This is not yet a published release.

## Phase 2.5 — release candidate

Independent/custom nutrition targets, reviewed calculated proposals/reset and collapsed editable Text Summary. See the [release verification and promotion record](PHASE_2_5_RELEASE.md) and [implementation history](PHASE_2_5_STATUS.md).

## Phase 3 — remaining roadmap

The [Phase 3 backlog](PHASE_3_BACKLOG.md) records the locally implemented catalogue/quota subset, remaining catalogue operational work and the original AI, voice, export and external-agent/MCP roadmap. Implementation sequence, AI providers, costs and privacy arrangements need approval.

## Phase 4 and later — conditional backlog

See [platform backlog](PHASE_4_BACKLOG.md): durable jobs/outbox, broader audit/operational analytics, notifications, offline/PWA, additional identity providers and scaling. No new server, Redis, broker, paid AI subscription or recovery service is authorized by these documents.

## Reading order and source authority

1. [Maintained feature list](1.%20FEATURE_LIST.md) and [maintained PRD](2.%20PRD.md).
2. [System design](3.%20SYSTEM_DESIGN.md), [database design](4.%20DATABASE_DESIGN.md), [API design](5.%20API_DESIGN.md) and [visual summary](ARCHITECTURE_VISUAL_SUMMARY.md).
3. [Project continuity](6.%20PROJECT_CONTEXT.md), [deployment](DEPLOYMENT.md) and [nutrition references](NUTRITION_REFERENCE_SOURCE.md).

The original [feature-list PDF](1.%20Feature%20List.pdf) and [PRD PDF](2.%20PRD.pdf) remain historical baselines. Maintained Markdown records their later scope changes. Runtime/contracts establish implementation evidence; proposed schemas do not describe deployed functionality. OpenAPI currently formalizes only a subset of runtime routes.

Workflow: feature branch from `dev` → PR to `dev` → release PR to `main`, with local verification before promotion. Documentation edits alone do not authorize push, merge or deployment.

[Individual feature checks for Phase 2.1](PHASE_2_1_FEATURE_CHECKLIST.md) cover the refined UI, welcome guide and local/USDA accounting.
