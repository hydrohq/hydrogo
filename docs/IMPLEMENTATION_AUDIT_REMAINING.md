# Implementation Audit — Remaining Production Work

Date: 2026-09-12

This document records audit findings that cannot be honestly closed by a local
source change alone. A checked item requires evidence from the target customer
environment, an external provider, legal review, or a release pipeline.

## Closed in source

- Operational telemetry no longer creates a visitor cookie or hashes an IP.
- Unique-visitor reporting is explicitly unavailable until the consent-gated
  analytics module exists.
- The login page no longer exposes a fixed demo email or password.
- Public GraphQL requests have a bounded JSON body, strict decoding, query field
  and depth budgets, and redacted public error messages.
- Stripe fallback requests use a timeout and reject redirects; provider error
  bodies are not copied into domain errors; oversized webhook bodies fail.
- Public GraphQL, checkout and payment-intent operations use bounded rate
  limits. Redis/Valkey is used when configured, with a bounded local fallback
  and a short fail-closed Redis operation timeout.
- Webhook URL structure is rejected at configuration load time when it contains
  credentials, fragments, opaque URLs or a missing host.
- Media, site, news routes and the news worker are registered only when their
  modules are enabled. GraphQL is not exposed without a headless content module.
- CSP no longer trusts Google font origins.
- Browser E2E uses a pinned Playwright/npm lockfile; CI always uploads trace,
  screenshot and video artifacts when a browser job fails.

## Still required before a production claim

### Architecture and module isolation

- Complete lazy construction for every module-owned store, cache, service,
  asset directory and migration. Add SQLite/PostgreSQL tests proving that a
  disabled module creates no route, table, worker, asset or mandatory resource.
- Split the composition root in `cmd/hydro/main.go` into boot-plan, route and
  worker composition packages.
- Move demo registry actions, demo seed records and the root redirect out of
  the generic product runtime.

### Privacy and product analytics

- Implement the optional `analytics` module from ADR-0022: explicit consent,
  purpose version, revocation, daily aggregates, retention and no event after
  rejection/revocation.
- Add legal review and versioned privacy/cookie/analytics pages before enabling
  analytics for a customer.

### Operations and release

- Execute a clean-environment PostgreSQL restore, media restore, session revoke,
  write-fence and job quarantine drill; record measured RPO/RTO and retention.
- Execute a real reverse-proxy Blue/Green rollout and rollback after a new write.
- Configure external backup/S3-compatible media, KMS/key escrow, alert delivery,
  SMTP bounce handling and production secret/key ceremony.
- Publish signed release provenance, SBOM and the release artifact from CI.

### Payments and membership

- Run Stripe, iyzico and PayTR sandbox/replay/refund/idempotency tests with real
  provider accounts and documented timeout behavior.
- Complete member email verification, password reset, consent/legal linkage,
  deletion and export. Enforce member-session access through a centralized
  route guard rather than relying only on individual handlers.

### Quality and product readiness

- Add risk-weighted coverage thresholds for auth SQL stores, member transport,
  payment providers and GraphQL error paths.
- Add accessibility checks for keyboard navigation, focus management, labels,
  contrast and reduced motion.
- Replace hard-coded demo/home/locale links with site settings and empty states.
- News now has a module-owned relational taxonomy model/repository and
  SQLite/PostgreSQL migration (`202609160001_news_taxonomy`), with SQLite and
  WSL PostgreSQL contract evidence in `R6-news-taxonomy-contract-2026-09-16`;
  admin editing, authorization integration and a real admin localization catalog
  remain open.
- Admin Studio now has a reusable, escaped taxonomy editor primitive
  (`internal/adminstudio/taxonomy.go`; ADR-0127); News route integration and
  module-level authorization are now wired through the protected News taxonomy
  endpoint, with transaction-backed replacement when the executor supports it;
  protected term catalog CRUD now validates full parent trees and refuses
  deleting terms with children or content relations; published News detail
  pages now show bounded taxonomy-aware related content with a safe fallback.
- The current revision also passed the complete WSL2 `go test -race ./... -p 1
  -count=1` suite; multi-process load and long-duration soak remain production
  gates (`docs/validation/R8-wsl-race-2026-09-16.md`).
- News now accepts an optional `extension/v1.EditorialWorkflow` and enforces
  transitions inside its transaction boundary, with a rejection regression
  test. Admin Studio hides disallowed publication actions using the same policy.
  Persisted custom moderation states, role-specific policies and workflow
  editing remain open.
- SDK contracts for taxonomy, forms, editorial workflow and search are now
  available under `extension/v1` (ADR-0120–0123); the Site Edition form inbox,
  status transitions and retention purge are implemented and audited. A
  schema-driven form-builder Admin Studio UX and persistent audited CRUD wiring
  are now available when the forms capability is enabled; external provider
  indexing remains open;
  the optional analytics capability now has an aggregate-only consent store;
  legal retention and DPO review remain open; SDK v1 technical support and
  deprecation policy is now documented, while commercial terms remain open;
  a bounded local search provider is available for Compact/prototype use
  (`internal/search.MemoryProvider`, ADR-0128).
- Validate the stable SDK, support window, compatibility policy and customer
  portability benchmark on real hardware.

The authoritative production gate list remains
`docs/OPEN_GATES_AND_OPERATOR_ACTIONS.md`; this file is the implementation
audit companion and must be updated when any item gains verifiable evidence.
