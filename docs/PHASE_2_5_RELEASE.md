# Phase 2.5 release candidate — 22 October 2026

## Phase 1 / Phase 2 baseline

The v1.0.0 and v2.0.0 releases remain historical code checkpoints. This release is additive; accepted targets, logged nutrition and health history remain intact. A code tag is not a database backup or a data-recovery guarantee.

## Phase 2.1 and Phase 2.5 included changes

- Shared spacing, themed dialogs and inline validation; amount-correction review and generic scheduled tasks.
- Local-first shared food catalogue with explicit USDA discovery, ten searches per user per UTC day, source deduplication and isolated usage accounting.
- Skippable account-persisted setup/feature guide and precise missing-profile-field navigation.
- Independent selected nutrition targets, exact 4/4/9 calorie refresh, initial calculated proposals, explicit adult calculated reset, source/assumption guide and immutable revision history.
- Applicable adult macro-range comparisons and server-bound confirmation; saved outside-reference labels remain visible.
- Shared collapsed editable Text Summary on Dashboard/Analysis, always-visible full Copy/Regenerate, draft/navigation guards, privacy-safe bounded partial-data reporting.
- Shared themed Analysis calendar with month/year selection and responsive positioning.

## Verification

- Web TypeScript and production build; 31 frontend regression tests, including clipboard-success/denial paths.
- API Go suite and Core Go suite against an isolated Mongo replica set. Coverage includes exact decimals, conversion/defaults, owner isolation, stale previews, signed/forged/expired/replayed target confirmations, initial proposal accept/decline lifecycle, summary privacy/pagination, malformed-section partial results and the 10,001-group bound.
- Both local HTTP smoke scripts pass using disposable synthetic accounts. Provider tests use cached fixtures, not live USDA requests.
- Chrome extension checks: initial target proposal, explicit acceptance, custom outside-range warning, Cancel preserving the draft, Confirm saving only edited fields, persistent confirmed label after session restoration, 390-pixel dialog with no horizontal overflow, full edited-summary copy, regeneration cancellation and unsaved-navigation guard. Theme/viewport restored.
- Formal OpenAPI/protobuf validation and whitespace checks pass. CI now includes all frontend regressions and non-skipped Mongo integration tests.

## Known limits and Phase 3

- Desktop Chrome checks are not a claim of Safari, native mobile, every assistive-technology combination or every network-failure permutation coverage. Clipboard permission denial has a deterministic regression; full manual-copy fallback remains available.
- Vite reports a non-blocking bundle-size warning (~537 kB uncompressed). Code splitting is a performance follow-up.
- Activity/protein mapping is explicitly an application approximation, not medical advice. Unknown age/sex is not invented. Water remains optional and independent; logged weight does not become profile weight.
- Catalogue maintenance/storage operations, broader provider budgets, and the separately approved Phase 3 roadmap remain separate. No barcode, AI, notification, offline/PWA or data-recovery feature is added here.

Additional Chrome session checks: sign out and sign in as a different disposable account showed a fresh factual summary with no previous draft. Expiring the shared disposable session while another tab had an edited summary redirected that tab to sign-in and removed the private draft. Session restoration succeeds on the configured `http://localhost:5173` origin; using an unconfigured port correctly fails origin-protected refresh.

## Promotion record

Feature PR: [#19 — Phase 2.5](https://github.com/architsangal/AteThat/pull/19), targeting `dev`. CI provisions MongoDB integration tests and all 31 web regressions. Production promotion and v2.5.0 publication follow only after protected checks. Actual deployment evidence is included in the GitHub release notes, which are published after production verification.
