# HydroGo Professionalization Backlog

This backlog outlines required operational hardening and verification tasks. Green local test runs do not imply external production gates are cleared; each item must be tracked with evidence, tests, and ADRs where applicable.

---

## P0 — Mandatory Pre-Production Gates

### Deployment & Rollback
- [ ] Produce reproducible OCI / Docker images for staging and production.
- [ ] Complete config/secrets segregation; verify secrets never leak into logs, dumps, or SBOMs.
- [ ] Rehearse two-process blue/green deployments behind a live reverse proxy.
- [ ] Connect readiness, warmup, asset retention, and schema compatibility gates to deploy pipelines.
- [ ] Execute rollback and previous-binary restart tests against real PostgreSQL databases following write traffic.
- [ ] Measure graceful shutdown, connection draining, and worker termination intervals.
- [x] Fix HTTP graceful shutdown to prevent returning before active handlers drain. (Evidence: `internal/server/server_test.go`.)
- [ ] Document automated rollback, failed migration abort, and human approval steps in operational runbooks.
- [x] Pin supported OS, Go, PostgreSQL, Redis/Valkey, and FFmpeg versions in `SUPPORT_MATRIX.md`.

### PostgreSQL & Data Integrity
- [ ] Execute the full contract suite against supported PostgreSQL versions in CI using live DSNs.
- [ ] Test migration and rollback against staging copies of production datasets.
- [ ] Take encrypted off-server backups and restore to clean target instances.
- [ ] Verify post-restore session revocation, job quarantine, egress fencing, and erasure replay.
- [ ] Measure RPO, RTO, retention, backup encryption, and key recovery workflows using production-scale data.
- [ ] Validate connection pool sizing, transaction timeouts, lock timeouts, and statement timeouts across capacity profiles.
- [ ] Document and test `CREATE INDEX CONCURRENTLY` and expand-migrate-contract workflows for large tables.

### Security & Identity
- [ ] Cover MFA, invitation, password reset, account recovery, session revocation, and CSRF flows with end-to-end browser tests.
- [ ] Test Redis rate limiting resilience under connection dropouts, failovers, clock skew, and key-space exhaustion.
- [x] Enforce fail-closed behavior on the Redis rate limiter when configuration or network connectivity is missing. (Evidence: `internal/security/redis_ratelimit_test.go`.)
- [ ] Test against upload abuse, SSRF, path traversal, decompression bombs, and oversized request bodies.
  - [x] Decoded image pixel budget has regression coverage; SSRF and decompression bomb tests remain open.
- [x] Harden Auth `return_to` redirects using URI-aware origin validation, rejecting encoded slash/backslash open-redirect variants. (Evidence: `internal/security/redirect.go`.)
- [x] Resolve fragment spreads and cut cyclic fragments fail-safe in GraphQL complexity guard. (Evidence: `modules/api/graphql/handler.go`.)
- [x] Add bounded fuzz smoke tests for the GraphQL query budget parser. (Evidence: `modules/api/graphql/handler_test.go`.)
- [ ] Establish documented procedures for secret rotation, signing-key rotation, and emergency revocation.
- [ ] Complete formal threat modeling and external penetration testing.
- [ ] Verify security headers, CSP transition from report-only to enforce, and trusted origins behind a production proxy.
- [x] Pin application security headers with regression tests (CSP nonce, nosniff, frame/referrer/cross-origin policies, HSTS). (Evidence: `internal/security/headers.go`, `headers_test.go`.)
- [x] Reject secret-like artifacts (e.g. `.env.*`, private keys, PKCS#12) in release manifests. (Evidence: `internal/release/manifest.go`, `manifest_test.go`.)
- [ ] Enforce PII masking, audit log access permissions, and log retention policies.

---

## P1 — Reliability & Operability

### Observability
- [x] Protect external webhook and SMTP retry loops with bounded attempts and exponential backoff. (Evidence: `internal/webhooks/dispatcher.go`, `internal/mailer/retry.go`.)
- [ ] Implement request, actor, and session trace correlation while masking tokens, passwords, and PII.
- [ ] Standardize metrics across latency, error rates, throughput, connection pools, caches, queues, and disk usage.
- [ ] Provide Prometheus and OpenTelemetry export adapters and dashboard templates.
- [ ] Test alerting pipelines for backups, disk saturation, failed migrations, SMTP outages, dead-letter queues, and elevated error rates.
- [ ] Define formally tracked SLOs, SLIs, and error budgets.

### Performance & Capacity
- [x] Add overflow protection and regression tests for monetary calculations in Commerce. (Evidence: `modules/commerce/model.go`, `service.go`.)
- [x] Enforce bounded HTTP timeout profiles (read/write/idle timeouts) in development proxies.

### Design System & Frontend
- [ ] Standardize a single design token scale (typography, spacing, color, radius, elevation, motion).
- [ ] Unify component contracts for Button, Form, Field Error, Table, Pagination, Modal, Toast, Tabs, Empty State, and Loading skeletons.
- [ ] Bind Admin, Publisher, and Public themes to consistent accessibility standards.
- [x] Add responsive desktop, tablet, and mobile viewport E2E tests. (Evidence: `tools/e2egen/main.go`; Chromium 20/20 PASS.)
- [ ] Integrate automated keyboard navigation, focus management, reduced-motion, and contrast checks into CI.
- [x] Implement idempotent document-level event listeners across HTMX fragment lifecycles to prevent listener proliferation. (Evidence: `modules/pages/transport/http.go`, `internal/media/admin.go`.)
- [x] Compute asset cache-buster versions using content SHA-256 fingerprints; eliminate manual date-based versioning. (Evidence: `tools/assetmanifest`, `tools/assetguard`.)
- [x] Bind frontend build manifests to project generation inputs, producing deterministic route and selector contracts. (Evidence: `internal/projectgen`.)
- [x] Validate I18N locale allowlists using BCP-47 patterns, deduplication, and default locale verification. (Evidence: `internal/config/config.go`.)
- [x] Eliminate floating-point conversions in monetary representations across Commerce. (Evidence: `modules/commerce/money.go`, `money_test.go`.)
- [x] Pin vendor assets (Quill, HTMX, Inter) with license and hash verification manifests. (Evidence: `static/vendor/*/LICENSE`, CSP hashes.)

---

## P1 — Core and Module Boundaries

- [x] Classify all routes into public, authorized, or admin scopes with default-deny regression assertions. (Evidence: `internal/router/router_test.go`.)
- [x] Isolate Pages and News public read, admin validation, and optimistic locking fixtures. (Evidence: module transport/service tests.)
- [x] Verify that inactive modules do not register routes, execute migrations, load assets, or start background workers. (Evidence: `cmd/hydro` inactive module tests.)
- [x] Enforce module boundary hygiene using `moduleguard` and `importguard`. (Evidence: `tools/moduleguard`, `tools/importguard`.)
- [x] Version capability contracts and test for deprecation and backwards compatibility. (Evidence: `extension/v1/version.go`.)
- [x] Verify public APIs never leak draft, scheduled, or soft-deleted records across any module. (Evidence: Pages, News, Site public projection tests.)
- [x] Test concurrent edits, idempotent jobs, outbox replays, and provider timeout edge cases. (Evidence: Pages optimistic locking, queue idempotency, outbox replay, SMTP retry tests.)

---

## P2 — Developer Experience & Quality

- [x] Make `hydro doctor` diagnostics actionable, safely reporting missing databases, caches, unwritable paths, and configuration drifts. (Evidence: `cmd/hydro/main.go`.)
- [x] Provide local dependency bootstrap and teardown workflows. (Evidence: `scripts/bootstrap-wsl.sh`, `compose.dev.yaml`, `docs/LOCAL_BOOTSTRAP.md`.)
- [x] Test scaffolding of all preset templates into temporary roots via `hydro new`. (Evidence: `scripts/generator-smoke.ps1`.)
- [x] Install pre-commit and pre-push hooks for formatting, vetting, and lint guards. (Evidence: `scripts/git-hooks/`.)
- [x] Integrate fuzz testing across parsers, sanitizers, paths, configurations, tokens, and webhooks.
- [x] Establish automated dependency auditing and SPDX SBOM generation. (Evidence: `.github/dependabot.yml`, SBOM tools.)
- [x] Prevent mutex copying in test fixtures and enforce race detection in CI. (Evidence: `go vet ./...`, `go test -race ./...`.)
- [x] Author comprehensive guides with examples for APIs, extensions, and modules. (Evidence: `docs/API_AUTHORING.md`.)

---

## Simplification & Pruning

- [ ] Strip unused vendor assets, unused theme variants, and demo fixtures from release images.
- [x] Remove manual date/cache version stamps in favor of content-hash fingerprints. (Evidence: `tools/assetmanifest`, `tools/assetguard`.)
- [ ] Enforce the Information Hierarchy to shield users from internal error codes, database driver details, and stack traces.
- [ ] Periodically prune unreferenced feature flags, dead routes, and obsolete migrations.
