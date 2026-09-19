# SaaS CMS Readiness Plan

**Status:** Active engineering plan. A checkbox is not complete until its stated
evidence exists in the repository or in a dated, reproducible environment report.

## Product boundary

HydroGo ships first as a managed **single-customer deployment**: one customer,
one database, isolated media namespace, and a versioned API. This is a sellable
managed CMS service, but not shared-database multi-tenant SaaS. Shared tenancy
is enabled only after the dedicated multi-tenant gates below pass.

## Delivery rules

- Every phase is one or more small commits; each commit passes its relevant
  focused tests and the final phase commit passes `go test ./...`, `go vet ./...`,
  and `git diff --check`.
- A remote push follows every completed phase. If the local hook cannot run,
  equivalent checks and the reason for an explicit `--no-verify` push are recorded.
- New or changed cross-layer decisions require an ADR, SQLite/PostgreSQL contract
  coverage, migration metadata, and deployment/cache impact notes.
- A real-environment drill is never replaced by a unit test or a prose claim.

## Phase 0 — Baseline hardening

- [x] Wire mobile bearer authentication, bounded JSON inputs, device revocation,
  refresh rotation, and mobile CSRF boundary.
- [x] Correct access-token expiry and make refresh rotation transactional.
- [x] Align deployment templates with typed runtime configuration and secret files.
- [x] Replace fabricated certification claims with pending drill evidence.

**Evidence:** commit `4a995a2`; full Go test/vet suite; source-level deployment review.

## Phase 1 — Truthful scope, release and evidence inventory

- [x] Reconcile README, roadmap, ADR status and validation reports with actual
  implementation; remove unsupported production/SaaS claims. The current
  source-to-document matrix is recorded in `R76-documentation-traceability-2026-09-19.md`.
- [x] Create a release gate manifest that fails when a claimed profile lacks
  database, restore, proxy, capacity, or security evidence. The machine-readable
  manifest and validator are `docs/validation/release-gate-manifest.json` and
  `go run ./tools/releasegates`; the Standard profile remains explicitly a
  candidate until the external gates are evidenced.
- [x] Establish a versioned OpenAPI baseline and CI-compatible breaking-change
  check for staff mobile, consumer/member mobile-commerce and public content
  API surfaces. Admin mutation APIs still need their own published contract
  before a general API compatibility claim is made.
- [x] Add a dependency/SBOM/vulnerability review cadence with recorded owners.
  Dependabot, the scheduled `dependency-review` workflow, 90-day evidence
  retention, and the role-based evidence matrix are documented in
  `docs/DEPENDENCY_TRIAGE.md`.

**Exit evidence:** source-to-document traceability table, release gate command,
`docs/validation/R66-release-gate-manifest-2026-09-19.md`, and a clean issue list
categorized as code, environment, legal, or product decision. See
`R76-documentation-traceability-2026-09-19.md` for the current matrix.

## Phase 2 — Mobile API completion

- [x] Scope and document the staff mobile principal as a staff/admin account.
  Staff routes authenticate core `auth.User` actors; consumer/member routes are
  a separate approved `members.Member` contract and must never be presented as
  staff login.
- [x] Decide the customer mobile audience: the client application uses the
  separate `members.Member` identity and member-only mobile credentials for
  private account/commerce flows; published reader/catalog projections remain
  unauthenticated (ADR-0150). Staff/admin work remains web-only.
- [ ] Add PostgreSQL mobile-token contract tests, concurrent-refresh tests, device
  revoke/disable-user tests, rate-limit tests and audit-event assertions. SQLite
  lifecycle coverage is executable locally; member/staff PostgreSQL lifecycle
  and concurrent-refresh coverage is now present as conditional contract tests
  (`tests/contract/*mobile*_contract_test.go`) and requires a configured DSN to
  produce environment evidence. The ownership-scoped member order-detail query
  is additionally certified in
  `tests/contract/member_order_detail_contract_test.go` on SQLite and
  PostgreSQL; the scoped WSL race suite also passes for member/commerce and
  contract packages (`R72-member-commerce-race-2026-09-19`). Login and MFA
  rate-limit plus audit assertions
  are present, including member login/refresh/revoke audit assertions; real
  production sink delivery and retention certification remain open.
- [x] Publish OpenAPI endpoints and error schemas; add pagination, idempotency and
  deprecation behavior for every mobile resource. Staff and consumer/member
  authentication surfaces are now published and baseline-checked; business
  catalog, cart and member order-history projections are now published and
  baseline-checked; member order detail/payment-status projection is now
  published; provider webhook delivery remains a backend-only integration and
  still requires provider sandbox evidence.
- [x] Add platform-neutral iOS/Android client contract fixtures for staff and
  consumer/member audiences, covering login, refresh, logout, offline retry,
  token loss and member cart/order scope.
- [ ] Run threat-model, abuse and device-loss drills with real deployment settings.

**Exit evidence:** ADR-0139 release gates, two dialects, authenticated E2E API
fixture, and dated mobile threat-model report.

## Phase 3 — Headless web frontend and BFF boundary

- [x] Select one official frontend reference implementation and separate it from
  HydroCore runtime code: `examples/headless-starter` is an independent
  TypeScript client restricted to the versioned API and covered by its client
  contract smoke in the browser-E2E workflow.
- [ ] Implement same-site proxy and BFF reference topologies; forbid browser
  credential forwarding across an untrusted origin. The scoped BFF request
  policy, bounded proxy and single-instance session/CSRF adapter are in
  `internal/security/bffpolicy`, `internal/security/bffproxy` and
  `internal/security/bffsession` (R68/R75/R79); a shared production session
  store/login exchange and staging E2E are still required.
- [ ] Prove public CORS allowlists, cross-origin mutation rejection, preview
  confidentiality, SEO metadata, sitemap, locale and media behavior end to end.
- [ ] Run frontend-to-API contract compatibility checks in both old/new API order.

**Exit evidence:** ADR-0020/0021 acceptance tests, browser E2E in staging, and a
versioned frontend compatibility matrix.

## Phase 4 — Production Standard profile certification

- [ ] Validate actual Docker/Compose or IaC deployment using PostgreSQL, Redis/
  Valkey, object storage, SMTP and a publicly trusted TLS certificate.
- [ ] Run two-process blue/green deploy, rollback-after-write and retained-asset
  drills behind the selected reverse proxy.
- [ ] Perform clean-target database-plus-media restores, session revocation,
  egress quarantine, key recovery and backup retention drills.
- [ ] Connect alert delivery to an owned on-call path and test detection-to-close.
- [ ] Measure representative capacity, p95/p99, RPO/RTO and cost envelopes.

**Exit evidence:** completed drill reports with command outputs and artifacts;
not templates, synthetic assertions or `main` as a commit identifier.

## Phase 5 — CMS product completion

- [ ] Define supported CMS bundles (corporate, editorial, commerce) and enforce
  inactive-module route/migration/worker/asset isolation for both SQL dialects.
- [ ] Finish editorial taxonomy, lifecycle, moderation, search, accessibility,
  locale and data-portability acceptance criteria for each sold bundle.
- [ ] Treat payments, analytics, OIDC/SAML and provider integrations as separately
  certified capabilities, never implied by a bundle label.
- [ ] Complete audit, retention, erasure/export and privacy operator workflows.

**Exit evidence:** edition acceptance fixtures, operator runbooks, module isolation
reports and a published support matrix.

## Phase 6 — Shared multi-tenant SaaS

- [ ] Turn ADR-0149 into an accepted implementation ADR only after the customer,
  pricing, residency and tenant lifecycle decisions are approved.
- [ ] Implement tenant context at the router, application service, repository,
  cache, jobs/outbox, media, audit, export/restore and authorization boundaries.
- [ ] Enforce fail-closed tenant scope and PostgreSQL RLS defense in depth.
- [ ] Implement tenant provisioning, suspension, deletion, quotas, billing events,
  super-admin auditing and per-tenant key/data lifecycle.
- [ ] Prove isolation through property tests, IDOR tests, cache/media/job leakage
  tests, restore/export tests and noisy-neighbor failure/load tests.

**Exit evidence:** ADR-0140 gates, PostgreSQL certification and a separate
multi-tenant production readiness decision. The current mock test is not evidence.

## Phase 7 — Distributed profile and commercial launch

- [ ] Certify multi-instance web/workers, Redis loss/reconnect, cache generation
  reconciliation, object storage/CDN failure and queue backpressure.
- [ ] Finalize release signing, provenance, support/deprecation policy, commercial
  licensing, DPA, incident communication and customer migration terms.
- [ ] Complete a bounded pilot with production incident, support, upgrade and
  restore evidence before advertising an SLA.

**Exit evidence:** ADR-0142 topology certification, signed release manifest,
pilot closure report and explicit profile-specific SLA/support decision.

## Go framework direction

Do not replace `net/http` with Gin or emulate inheritance-based OOP. HydroGo's
framework surface remains composition-root wiring, `http.Handler` adapters,
small interfaces, module contracts and versioned SDKs. A framework replacement
requires a benchmark and migration ADR showing a measurable benefit that exceeds
the compatibility, test and supply-chain cost.
