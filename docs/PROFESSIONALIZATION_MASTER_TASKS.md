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

- [x] Add a consistent `hydro dev` command with bounded reload/error output by
  delegating to the existing local-only devserver supervisor.
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
- [x] Add actionable command help, examples, and failure remediation text;
  workflow guidance lives in `docs/CLI_WORKFLOW.md`.
- [x] Add and validate `examples/module-starter` as a first-module tutorial
  fixture; clean-temp generated output remains covered by generator smoke.

### EXT-01 Extension SDK boundary (`LOCAL`)

- [x] Publish a capability/version matrix for Core, modules, themes, and APIs
  in `docs/SDK_CAPABILITY_MATRIX.md`.
- [x] Separate stable SDK imports from internal implementation imports for the
  module route boundary via `extension/v1`; broader capability extraction stays
  in the SDK v1 task.
- [x] Add compile-time compatibility and deprecation coverage for the public
  extension contracts; legacy v0 types now point to `extension/v1`.
- [x] Add optional module lifecycle contracts for install, enable, disable,
  upgrade, and migration preflight in `extension/v1`.
- [x] Enforce that optional admin site/media routes are registered only when
  their modules are enabled; inactive modules now add no corresponding routes.
- [x] Reject duplicate route patterns (including generated API v1 aliases) with
  a startup error instead of allowing `ServeMux` to panic.
- [x] Add dependency graph validation and cycle detection for modules (see
  the versioned SDK boundary contract below).
- [x] Add dependency graph validation and cycle detection for modules in the
  versioned SDK boundary; runtime resolution remains Core-owned.
- [x] Add signed/allowlisted package metadata design without introducing a
  marketplace dependency in `docs/MODULE_PACKAGE_METADATA.md`.

### DATA-01 Database and repository contracts (`LOCAL`)

- [x] Expand SQLite repository contract fixtures and cross-dialect error mapping;
  SQLite behavior is certified locally and real PostgreSQL execution remains an
  external gate.
- [x] Normalize member uniqueness errors at the SQL repository boundary for
  SQLite and PostgreSQL, preserving unrelated driver errors.
- [x] Centralize SQL uniqueness detection in the infrastructure adapter so
  repository conflict semantics do not drift between modules.
- [x] Add local query-count/N+1 regression fixtures for page and news list
  projections; PostgreSQL planner and customer-sized latency evidence remain
  external gates.
- [x] Add bounded public/admin list indexes with a local planner regression
  fixture; PostgreSQL planner and production sizing remain external gates.
- [x] Enforce the same 1–100 list limit at page/news repository boundaries,
  protecting direct callers that bypass HTTP validation.
- [x] Add local profile-based pool and transaction contract tests; PostgreSQL lock and statement timeout certification remains an external gate.
- [x] Add portable export/import coverage for relations, locales, revisions,
  and secret/session exclusion; SQLite is certified locally and PostgreSQL
  execution remains an external gate.
- [x] Add migration metadata linting for compatibility, rollback, and lock cost
  via `tools/migrationguard`.

## P1 — Security, reliability, and frontend productization

### SEC-01 Security hardening (`LOCAL`)

- [x] Build a reusable abuse corpus for upload, SSRF, traversal, decompression
  bombs, oversized bodies, and malformed multipart input. Coverage spans
  `internal/security`, `internal/media/upload_test.go`, and
  `internal/media/http_test.go`; external penetration testing remains open.
- [x] Add secret/key rotation and emergency revoke runbook plus local contract
  tests; real key ceremony and secret-manager execution remain external gates.
- [x] Add PII masking and audit access-policy tests for logs and admin views;
  unauthorized member detail requests are denied and optional session storage
  cannot panic the admin read path.
- [x] Add bounded CSP report parsing at `POST /csp-report` with malformed,
  oversized, and CSRF-bypass regression coverage; reports are not persisted or
  logged.
- [x] Expand boundary tests so raw SQL/cache/queue details cannot leak into
  domain/application packages.
- [x] Add an import guard regression suite that rejects concrete SQL, Redis, and
  queue adapters in module application packages.
- [x] Redact browser transport error details on member and admin pages;
  diagnostics remain server-side and users receive stable generic messages.
- [x] Harden admin blank-target links with `noopener noreferrer` to prevent
  opener access across browsing contexts.

### SEC-02 Real security gates (`EXTERNAL`)

- [ ] Run external penetration test and threat-model review.
- [ ] Validate HTTPS, proxy trust, secure cookies, HSTS, and CSP enforcement in
  the production topology.
- [ ] Test Redis/Valkey failure, failover, clock skew, and multi-instance rate
  limiting with a real service.

### REL-01 Observability and operations (`LOCAL`)

- [x] Standardize metric names, labels, cardinality limits, and log fields in
  `docs/OBSERVABILITY_CONTRACT.md`.
- [x] Add request/actor/session/job correlation with mandatory secret/PII
  redaction.
- [x] Add the dependency-free Prometheus-compatible exporter behind explicit
  `Metrics` configuration; OpenTelemetry remains a separate integration gate.
- [x] Add deterministic alarm rules and dashboard examples for DB, cache,
  queue, media, backup, migration, SMTP, and dead-letter conditions in
  `docs/ALARM_RULES.md`.
- [x] Add incident, rollback, and degraded-mode runbook templates in
  `docs/runbooks/INCIDENT_RESPONSE_TEMPLATE.md`.
- [x] Add target-neutral SLO/SLI configuration schema and validation in
  `internal/ops` (targets remain optional and operator-owned).
- [x] Execute the full Go race suite in the supported WSL toolchain; Windows
  hosts without a C compiler are directed to the WSL path in validation docs.

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
- [x] Extract shared Button, Field, FormError, Table, Pagination, Empty,
  Loading, and Error components in `internal/ui`; Modal, Toast, and Tabs remain
  gated on a browser lifecycle/focus contract in `docs/UI_COMPONENTS.md`.
- [x] Harden the shared metadata-driven Table primitive with an accessible
  caption and scoped column headers in `internal/adminstudio`.
- [x] Give metadata-driven lists an accessible empty state with a status role
  and correct table-column span.
- [x] Give the shared rich-text editor an explicit multiline textbox role and
  server-rendered accessible name.
- [x] Make HydroGo and shadcn visual modes consume the same semantic markup;
  shared selector coverage is guarded by an internal UI contract test.
- [x] Add local keyboard/focus, reduced-motion, and accessible-name regressions; browser contrast and device audits remain external UI gates.
- [x] Add visual regression baselines for desktop, tablet, and mobile (seeded local Playwright goldens; browser/device approval remains a release review).
- [~] Add RTL/long-text/locale browser fixtures and locale-aware date, number and currency helpers; translated copy and full product locale matrix remain product-owned.
- [x] Add safe locale-derived RTL/LTR document direction and Unicode locale
  regressions; translated copy and formatting matrix remain product/browser
  gates.
- [x] Carry the bounded direction through the v1 theme page contract with a
  left-to-right default for legacy theme callers.
- [x] Add a theme render fixture for RTL locale attributes and long Unicode
  content; full locale formatting/browser matrix remains open.
- [x] Verify auth, editor, preview, media picker, cart, and checkout through
  refresh/back-forward browser scenarios. Cross-origin proxy/BFF behavior
  remains a separate product/ADR gate. Chromium run: 35/35 PASS.

### UI-02 Frontend architecture decision (`LOCAL` + `PRODUCT DECISION`)

- [x] Document when to use server-rendered HTML, HTMX, or a separately built
  frontend in `docs/FRONTEND_ARCHITECTURE.md`.
- [x] Define a versioned frontend manifest/route/selector contract in
  `internal/ui` and `docs/FRONTEND_MANIFEST.md`.
- [x] Define state, error, loading, and optimistic-update conventions in
  `internal/ui` and `docs/FRONTEND_STATE_CONVENTIONS.md`.
- [x] Add a repeatable targeted/full-compile iteration baseline command in
  `scripts/measure-iteration.ps1`; pilot feature delivery and deployment time
  remain customer evidence gates.

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
- [~] Provider-neutral S3-compatible media boundary and optional local MinIO
  profile implemented (ADR-0022); production provider/IAM/CDN validation,
  retention and failover remain deployment gates.

## P1 — Prototype velocity and developer experience (`LOCAL`)

The detailed execution plan is in
[PROTOTYPE_VELOCITY_PLAN.md](PROTOTYPE_VELOCITY_PLAN.md). These tasks preserve
the existing Core/module boundaries:

- [~] `hydro dev` one-command local stack with reload, seed and dependency lifecycle; Compose Redis/MinIO auto/required/none modes are implemented, while browser auto-open and mail preview remain optional local conveniences.
- [~] Schema-driven resource/CRUD/page/API/admin/form generation from one reviewed contract; generated artifacts and aliases are available, runtime wiring remains an explicit review step.
- [~] Provider-neutral design-system adapters (native/shadcn/material/carbon metadata), authenticated component showcase, responsive visual regression baselines and opt-in RTL/locale overflow fixtures are available; translated copy and product-owned locale approval remain open.
- [~] News, ecommerce, corporate, membership, dashboard and ERP starter presets are available; deeper seed/upgrade smoke coverage remains open.
- [~] Typed API client, OpenAPI, form, pagination/filter/sort query and cache-key invalidation templates are generated; server-side policy and runtime cache wiring remain explicit module work.
- [~] Generated bounded bulk-selection, progress and CSV import/export primitives are available; module-specific upload handlers and optimistic persistence remain explicit runtime work.
- [~] Iteration measurement script and generator compatibility check (`hydro validate generator`) are available; full telemetry, ADR drift checks and upgrade assistant remain open.
- [x] Benchmark acceptance gate is automated in `scripts/measure-iteration.ps1` (first scaffold, route contract, CRUD artifact and compatibility) and runs on the clean Windows CI build.

### PROD-03 Deliberately gated expansion (`EXTERNAL`)

- [ ] Decide OIDC/SAML, mobile API, multi-tenant, AI, and Distributed scope from
  customer evidence and separate ADRs.
- [ ] Select SDK v1 support window, LTS, SLA, licensing, and deprecation policy.

## P3 — Release hygiene and simplification (`LOCAL`)

- [x] Complete a release-hygiene audit for unused vendor assets, demo
  endpoints, dead routes, duplicate helpers, and inaccessible migrations;
  no safe deletion candidate was found in the current revision. Evidence:
  `docs/validation/release-hygiene-audit-2026-09-15.md`.
- [x] Guard embedded runtime vendor JS/CSS assets against disappearing
  references; reviewed deletion and broader dead-route audit remain separate.
- [x] Keep demo/fixture files out of the documented production image stage and
  require explicit `seed-demo --confirm-demo` for local data injection.
- [x] Add reproducible build checks and evidence guidance; external signing,
  registry provenance, and retention remain deployment gates in
  `docs/REPRODUCIBLE_RELEASES.md`.
- [x] Add dependency/license/vulnerability triage cadence documentation in
  `docs/DEPENDENCY_TRIAGE.md`.
- [x] Add a release checklist that requires all P0 evidence and explicitly lists
  unresolved external gates in `docs/RELEASE_CHECKLIST.md`.

## Current evidence baseline

The repository currently has passing local full tests, shuffled tests, fuzz
smoke, vet, vulnerability scan, generator smoke, edition boundary checks,
health/readiness smoke, module gating, security headers, and browser E2E
contracts. The remaining production and product-decision gaps are tracked in
[OPEN_GATES_AND_OPERATOR_ACTIONS.md](OPEN_GATES_AND_OPERATOR_ACTIONS.md) and
must not be silently closed.
