# HydroGo Professionalization Master Task List

**Created:** 2026-09-14  
**Owner:** HydroGo Core team  
**Scope:** developer experience, extensibility, frontend productization,
reliability, security, operations, and product boundaries.

This is the execution index for making HydroGo useful as a general-purpose,
developer-friendly platform. It complements the detailed
[Professionalization Backlog](PROFESSIONALIZATION_BACKLOG.md) and the
[Roadmap](ROADMAP.md). A task is not considered complete because its code
compiles: the task must have the appropriate tests, documentation/ADR, and
environment evidence. `LOCAL` means it can be implemented and proven in this
repository. `EXTERNAL` means it needs a real service, customer environment,
credential, legal decision, or product decision and must remain an open gate.

## Execution rules

- Work top-to-bottom within each priority unless a dependency forces a change.
- Keep Core free of customer-specific behavior; use modules and Client Editions.
- Inactive modules must create no routes, migrations, assets, workers, or
  mandatory resource overhead.
- Every architecture/security/data contract change gets an ADR and validation
  report.
- Every meaningful phase gets targeted tests, full local verification, a small
  commit, and a push.
- Never mark an external gate complete using a local mock alone.

## P0 — Local foundation before broad platform work

### DX-01 CLI and project workflow (`LOCAL`)

- [ ] Add a consistent `hydro dev` command with bounded reload/error output.
- [x] Add `hydro module create` with model/service/repository/transport,
  migration, deny-by-default policy, fixture/contract tests, admin manifest,
  and module metadata scaffolds. Business rules remain explicit review work.
- [x] Add `hydro resource create` for a collection and generated CRUD contract
  (shared, reviewable module generator).
- [x] Add `hydro migration create` with dialect files and metadata skeleton;
  checksums and SQL remain explicit review steps.
- [x] Add dependency-free `hydro validate [--json]` with deterministic module
  output; runtime checks remain under `hydro doctor`.
- [x] Add bounded `hydro test` wrapper with direct process arguments, timeout,
  package selection, race toggle, and stable errors.
- [ ] Add actionable command help, examples, and failure remediation text.
- [x] Add and validate `examples/module-starter` as a first-module tutorial
  fixture; clean-temp generated output remains covered by generator smoke.

### EXT-01 Extension SDK boundary (`LOCAL`)

- [x] Publish a capability/version matrix for Core, modules, themes, and APIs
  in `docs/SDK_CAPABILITY_MATRIX.md`.
- [x] Separate stable SDK imports from internal implementation imports for the
  module route boundary via `extension/v1`; broader capability extraction stays
  in the SDK v1 task.
- [ ] Add compile-time compatibility and deprecation tests for every public
  extension contract.
- [ ] Add module lifecycle contracts: install, enable, disable, upgrade, and
  migration preflight.
- [ ] Add dependency graph validation and cycle detection for modules.
- [x] Add dependency graph validation and cycle detection for modules in the
  versioned SDK boundary; runtime resolution remains Core-owned.
- [ ] Add signed/allowlisted package metadata design without introducing a
  marketplace dependency.

### DATA-01 Database and repository contracts (`LOCAL`)

- [ ] Expand SQLite/PostgreSQL repository contract fixtures and error mapping.
- [ ] Add query-plan/N+1 detection fixtures for list/detail surfaces.
- [ ] Add profile-based pool, transaction, lock, and statement timeout tests.
- [ ] Add portable export/import coverage for relations, locales, revisions, and
  secret/session exclusion.
- [ ] Add migration metadata linting for compatibility, rollback, and lock cost.

## P1 — Security, reliability, and frontend productization

### SEC-01 Security hardening (`LOCAL`)

- [ ] Build a reusable abuse corpus for upload, SSRF, traversal, decompression
  bombs, oversized bodies, and malformed multipart input.
- [ ] Add secret/key rotation and emergency revoke runbook plus local contract
  tests.
- [ ] Add PII masking and audit access-policy tests for logs and admin views.
- [ ] Add CSP report parsing and regression coverage for all generated pages.
- [ ] Expand boundary tests so raw SQL/cache/queue details cannot leak into
  domain/application packages.

### SEC-02 Real security gates (`EXTERNAL`)

- [ ] Run external penetration test and threat-model review.
- [ ] Validate HTTPS, proxy trust, secure cookies, HSTS, and CSP enforcement in
  the production topology.
- [ ] Test Redis/Valkey failure, failover, clock skew, and multi-instance rate
  limiting with a real service.

### REL-01 Observability and operations (`LOCAL`)

- [x] Standardize metric names, labels, cardinality limits, and log fields in
  `docs/OBSERVABILITY_CONTRACT.md`.
- [ ] Add request/actor/session/job correlation with mandatory secret/PII
  redaction.
- [ ] Add Prometheus/OpenTelemetry exporters behind explicit configuration.
- [ ] Add deterministic alarm rules and dashboard examples for DB, cache,
  queue, media, backup, migration, SMTP, and dead-letter conditions.
- [x] Add incident, rollback, and degraded-mode runbook templates in
  `docs/runbooks/INCIDENT_RESPONSE_TEMPLATE.md`.
- [ ] Add SLO/SLI configuration schema and validation (without inventing target
  values).

### REL-02 Capacity and deployment proof (`EXTERNAL`)

- [ ] Run real PostgreSQL contract and restore tests.
- [ ] Run two-process reverse-proxy blue/green, drain, rollback, and old-binary
  restart drills.
- [ ] Measure Compact/Standard/Distributed CPU, RSS, FD, pool, queue, latency,
  and error limits with customer-like data.
- [ ] Run 24-hour soak, spike, media, FFmpeg, and backup capacity tests.

### UI-01 Shared design system (`LOCAL`)

- [x] Define one token contract for type, spacing, color, radius, elevation, and
  motion across Admin, public themes, and member surfaces in
  `docs/UI_DESIGN_SYSTEM.md`.
- [ ] Extract shared Button, Field, FormError, Table, Pagination, Modal, Toast,
  Tabs, Empty, Loading, and Error components.
- [ ] Make HydroGo and shadcn visual modes consume the same semantic markup.
- [ ] Add keyboard, focus, reduced-motion, contrast, and accessible-name tests.
- [ ] Add visual regression baselines for desktop, tablet, and mobile.
- [ ] Add RTL, long-text, locale, date, number, and currency fixtures.
- [ ] Verify auth, editor, preview, media picker, cart, and checkout through
  refresh/back-forward and cross-origin browser scenarios.

### UI-02 Frontend architecture decision (`LOCAL` + `PRODUCT DECISION`)

- [ ] Document when to use server-rendered HTML, HTMX, or a separately built
  frontend.
- [ ] Define a versioned frontend manifest/route/selector contract.
- [ ] Define state, error, loading, and optimistic-update conventions.
- [ ] Measure build and iteration time for a standard customer feature.

## P2 — General product capabilities

### PROD-01 Content and workflow (`LOCAL` after product acceptance)

- [ ] Add taxonomy contract: categories, tags, multilingual labels, and related
  content.
- [ ] Add configurable editorial workflow and moderation transitions.
- [ ] Add form-builder capabilities with validation and inbox retention policy.
- [ ] Add search provider contract only after measuring real query needs.
- [ ] Add durable analytics/consent model only with a privacy ADR.

### PROD-02 Commerce and integrations (`EXTERNAL`)

- [ ] Validate Stripe/iyzico/PayTR sandbox callback, signature, idempotency,
  refund, and timeout behavior with real accounts.
- [ ] Decide and validate S3/CDN, durable queue, and Redis/Valkey production
  boundaries.

### PROD-03 Deliberately gated expansion (`EXTERNAL`)

- [ ] Decide OIDC/SAML, mobile API, multi-tenant, AI, and Distributed scope from
  customer evidence and separate ADRs.
- [ ] Select SDK v1 support window, LTS, SLA, licensing, and deprecation policy.

## P3 — Release hygiene and simplification (`LOCAL`)

- [ ] Remove unused vendor assets, demo endpoints, dead routes, duplicate
  helpers, and inaccessible migrations from release builds.
- [ ] Split demo/fixture data from production images.
- [ ] Add reproducible build and release artifact provenance checks.
- [ ] Add dependency/license/vulnerability triage cadence documentation.
- [ ] Add a release checklist that requires all P0 evidence and explicitly lists
  unresolved external gates.

## Current evidence baseline

The repository currently has passing local full tests, shuffled tests, fuzz
smoke, vet, vulnerability scan, generator smoke, edition boundary checks,
health/readiness smoke, module gating, security headers, and browser E2E
contracts. The remaining production and product-decision gaps are tracked in
[OPEN_GATES_AND_OPERATOR_ACTIONS.md](OPEN_GATES_AND_OPERATOR_ACTIONS.md) and
must not be silently closed.
