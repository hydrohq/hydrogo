# HydroGo application roadmap

**Date:** 2026-09-10 · **Status:** R0, R1 S03, and R2 completed; R3 security start gate completed, commercial pilot gates open.

This file tracks the order of work and delivery gates. Architectural rulings are in ADRs, detailed scope is in the [master plan](implementation_plan.md), and experiment definitions are in the [pre-implementation review](PRE_IMPLEMENTATION_REVIEW.md). Schedule estimation will be made after the initial technical experiments are completed. A gate only closes with a code revision and a validation report.

## 2026-09-10 reliability and security hardening progress

- Added a bounded failure counter and a real-time `hydro_readiness_state` gauge metric for readiness; validated state-based alarm recovery contract.
- CI, local, and production gate flows enforce bounded test timeouts, reproducible race runs, and immutable GitHub Action pins across all Go packages.
- Added `tools/actionguard`, Dependabot, CodeQL, dependency review, and a private security reporting policy.
- These advancements strengthen automated technical gates; true proxy rollback, clean environment restore, external key/backup recovery, and customer capacity/RPO/RTO evidence remain as open gates below.

## R0 — Preparation of repository and decisions

- [x] Define the initial route/publish and restore boundary with deletion history via ADR-0014.
- [x] Place SEO/redirects in pilot scope; keep route identity and publish atomicity in the first slice.
- [x] Separate Compact pilot tests from L2/Distributed tests.
- [x] Create starter folders, README, and contributing flow.

Output: development-ready source layout and this roadmap. Contains no working application or SDK.

## R1 — Technical experiments and walking skeleton

**Dependency:** R0. **Responsibility:** Core/SDK, DB, UI, build.

- [x] Create the first `go.mod` via the `github.com/ardaozyildirim/hydrogo` GitHub namespace; choose and pin Go/toolchain version.
- [x] S01: external Go module consumer fixture, allowed import boundary, and ensuring unselected modules do not enter the build.
- [x] S02/S04: SQLite/PostgreSQL driver and migration experiment; content/audit/outbox rollback with a single transaction scope.
- [x] S03: Templ, Quill, HTMX, and Alpine CSP compliance if necessary; two small themes and an upload experiment. (Local, versioned assets; CSP fixture with nonce; sanitizer, media, and external edition build evidence complete.)
- [x] Windows development/Linux amd64 build; availability of backup and image encoder tools.
- [x] Single `hydro` entry, config validation, health/readiness, graceful shutdown, default-deny router, bootstrap/session, and CSRF/audit foundation.
- [x] After Go tools are selected, add CI format/vet/test/build and two DB service matrices. (CI green; tests and dialect matrix running.)

**Output:** Build and entry in a clean environment; rollback in two DBs; external consumer build; CSP experiment. Dependency versions and alternatives are logged under `docs/validation/`. SDK remains v0.

## R2 — Pages and first customer theme

**Dependency:** R1. **Responsibility:** Pages, auth, media, UI.

First vertical slice pushed: Pages draft/revision/publish service, protected JSON routes, public published route, bounded process-local cache, and media upload with `pending → ready` metadata flow are ready. Preview, admin UI, and full output evidence are open.

- [x] Pin UUID/locale/revision, slug, and email normalization via a fixture before the first persistent schema. (Pages contract fixture covers locale/slug/revision and normalized route behavior.)
- [x] Pages create/edit/draft/preview/publish; admin and JSON should run over the same service/policy. (Protected JSON and admin form with CSRF use the same ContentService.)
- [x] Optimistic locking, record/field authorization, and conflict behavior of two editors. (Stale editor conflict and editor/admin mutation policy are validated within the contract.)
- [x] ADR-0014: draft URL must not alter live; publish route/generation/audit/outbox should be atomic. (Draft/published projection and duplicate route conflict contract tests pass.)
- [x] Local media pending/ready, private/public, sanitizer, and immutable revision body. (Upload metadata service and sanitized revision contract test are ready.)
- [x] First external Client Edition and typed view slots; no Core fork or internal imports. (examples/client-edition and import boundary CI evidence.)
- [x] First contract of structured Pages blocks and SQLite/PostgreSQL storage layer. (Versioned `hero`, `faq`, `cta` blocks, bounded payload validation, draft/revision/published projection transfer, and ADR-0016.) Admin block editor and visual renderer belong to the next slice.
- [x] Bounded local cache, public HTTP policy, and persistent outbox; no mandatory L2 for the first slice.

**Output:** The same end-to-end flow in two DBs; authorization and draft/private media leak tests; concurrent edit/publish conflict and crash rollback evidence. Evidence: [R2 validation report](validation/R2-pages-media-2026-09-08.md). Full SEO, sitemap, and historical URL management are in R3.

## R3 — Compact commercial pilot

**Dependency:** R2. **Responsibility:** Site Edition, security, release/ops.

R3 initial commit: disabled user/session revoke, admin-controlled user disable route, encrypted TOTP/recovery code, MFA-enforced login, hashed single-use password reset, request/confirm HTTP flow, admin invite/accept flow, and separate bounded login/MFA limiter foundation are ready. Compact SQLite snapshot/manifest and non-DB hash-chained erasure log intent experiments are successful. Evidence: [R3 MFA/session validation report](validation/R3-mfa-session-2026-09-08.md), [SQLite backup manifest experiment](validation/R3-backup-manifest-2026-09-08.md), and [external erasure log experiment](validation/R3-erasure-log-2026-09-08.md). Production notifier adapter, clean environment restore, PostgreSQL, erasure replay, and Compact capacity evidence are open.

- [x] Navigation, settings, SEO/canonical/Open Graph/sitemap, redirects, and limited form inbox. (Store, SQLite contract, DB-level navigation position uniqueness, public SEO/sitemap, admin mutation, allowlist/rate-limited contact submit, inbox status, retention purge, and in-transaction audit are ready; personal data access and runbook policy are defined in [PERSONAL_DATA_POLICY.md](PERSONAL_DATA_POLICY.md).)
- [x] MFA, invite/reset/recover, role/session revocation, and security fixtures. (TOTP/recovery, single-use reset, transactional invite accept, disabled-user session revoke, login/MFA limiter contracts, fail-closed shared-store limiter contract, STARTTLS/AUTH notifier adapter, bounded retry sender contract, and migration/contract test carrying HTTP request ID to audit event are complete; real SMTP delivery, provider bounce/alerting, and multi-instance store implementation are at the R3 ops gate.)
- [ ] S05-C: Stale refill, lost invalidation, cache bypass/revalidation, and bounded DB load in Compact `all` and blue/green transition. (Local public cache generation guard, in-flight bypass/revalidation on generation change, single-flight miss for the same key, 64 concurrent lookup → 1 repository call contract and benchmark, profile budget contract, concurrent Blue/Green traffic and cache invalidation simulation contract, and stale refill rejection are ready; real blue/green traffic and profile capacity measurement are open.) [Cache validation report](validation/R3-cache-generation-2026-09-08.md) and [capacity profile validation](validation/R3-capacity-profiles-2026-09-09.md). L2 outage is not a requirement for this gate.
- [ ] S06: online migration on the single supported server topology, old binary restart, rollback after a new write, and retained asset testing. (SQLite clean install/restart/no-op/checksum, fail-closed rejection of unknown forward migration by old binary, old binary rollback read-write, immutable retained asset copy with checksum validation from release manifest, `hydro upgrade check` preflight command, transactional/non-transactional migration runner based on metadata, PostgreSQL `CREATE INDEX CONCURRENTLY` short-read and long snapshot lock contracts, blue-green state machine enforcing readiness/warmup/asset-retention/schema-safety gates, and gated process-local HTTP slot switch/rollback contract are ready; reverse-proxy handler integration is validated at the contract level; real proxy/process restart drill and rollback after a new write are open.) [Migration validation report](validation/R3-migration-restart-2026-09-08.md)
- [ ] S07: DB+media restore, session revocation, job/egress quarantine, and old deployment write prevention. (Compact SQLite staging restore + control marker, `manifest.json` inside snapshot, `restore_controls`/write-fence migrations, session revocation on isolated PostgreSQL schema and SQLite, old deployment write rejection and job claim quarantine/release contracts, custom-format PostgreSQL dump/restore adapter, manifest-validated PostgreSQL media staging, atomic media activation requiring control marker, isolated target database round-trip test, and staging-first `hydro restore verify|stage|activate-media` CLI flow are ready; sold profile certification is open.) Local PostgreSQL 18.6 contract/restore evidence is recorded in the [R3 validation report](validation/R3-postgres-local-contract-2026-09-09.md); clean external target and sold profile restore drill are open.
- [ ] External erasure log adapter/checkpoint prototype for ADR-0014; post-backup deletion, crash, missing log, and retention tests. (Hash-chained JSONL, checkpoint replay with hash-anchor validation, idempotent append, retry replay after failure, SQLite and PostgreSQL Pages/media SQL adapter, idempotent GC contract using final pin check, and manifest-validated pin-aware local snapshot retention pruner are ready; post-backup deletion, real storage provider, and external provider drill are open.)
- [ ] Measure and fix resource limits, `max_stale`, session durations, RPO/RTO, retention, and rollback window. (Media/DB byte and duration budget API for SQLite snapshot, along with staging cleanup contract on exceedance, are ready; real customer RPO/RTO measurement and retention window are open.)
- [ ] Runbook, alarms, external backup/key recovery; finalize customer delivery/license/support terms pre-sale. (Initial operations runbook, `hydro doctor` checking profile/checksum/media/backup-tool/writability, SQLite/PostgreSQL staging-first backup/restore CLI, dry-run/confirm snapshot retention, protected bounded HTTP/request-latency/DB-pool/cache/jobs/worker metrics, and deterministic `internal/ops` alarm evaluator contract for backup/disk/migration/SMTP/dead-letter are ready; real clean environment drill, alarm delivery, and external key/backup evidence are open.)

**Output:** One real customer job, security/restore, and measured Compact capacity evidence. Each runbook has been executed in a clean environment. SDK is still v0; Distributed is not in sales scope.

## R4 — News and SDK validation

**Dependency:** R3. **Responsibility:** News, SDK, DB/release.

- [x] News review/schedule/publish/withdraw; immutable revision, schedule generation, and re-authorization check. (News migration, domain/repository, SQLite schedule generation contract, worker lease/idempotency, JSON/public routes, and protected admin HTML forms with CSRF are ready.) [R4 validation report](validation/R4-news-admin-2026-09-08.md)
- [x] Second external Client Edition, second theme, and different custom business rule. (examples/news-edition separate module, EditorialTheme, importguard/build CI step, and `EditorialDraft` approval/desk rule independent of Core are complete.) [R4 validation report](validation/R4-news-admin-2026-09-08.md)
- [x] Transaction, hook, event/job lease, and idempotency contract kits across two modules. (Pages/News transaction-audit/outbox patterns, bounded queue, SQLite/PostgreSQL compatible durable `jobs` lease store, durable queue capacity check serialized in transaction, News durable queue adapter, due scheduler discovery, bounded concurrency with unique worker IDs tied to profile budgets, production worker bootstrap, bounded exponential retry, dead-letter transition with lease validation, bounded error diagnostics, and bounded idempotent `internal/hooks.Dispatcher` with retry/dead-letter/concurrency metrics visibility are ready.) [Job store validation report](validation/R4-job-store-2026-09-08.md) · [Capacity validation report](validation/R4-job-capacity-2026-09-09.md)
- [x] Core/module/schema/config/content/event/job/asset compatibility matrix and legacy customer fixtures. (Proven epoch/blue-green boundaries gathered in [COMPATIBILITY_MATRIX.md](COMPATIBILITY_MATRIX.md); legacy customer upgrade fixture from migration 008 to full schema, published projection backfill, media and auth hash preservation are validated via contract test.)
- [x] Identify the proven small SDK surface as a v1 candidate; extract patterns for the generator. (Theme/Module typed contract under `extension/v1` and explicit candidate version are pinned.) [SDK validation report](validation/R5-sdk-v1-candidate-2026-09-08.md)

**Output:** Two modules + two separate Editions run without a Core fork; SDK v1 candidate is reviewed. The SDK part of S08 closes; the generator part closes in R5.

## R5 — Generator and stable v1

**Dependency:** R4. **Responsibility:** DX/SDK, data, release.

- [x] Deterministic CRUD generator, generated/custom protection, deny default, and reviewable dialect migration draft. (Collection spec generator, true CRUD repository/transport generation, secret/destructive deny, compilable Pages/News fixtures, published migration immutability, and reviewable dialect migration skeleton are ready.) [Generator validation report](validation/R5-generator-2026-09-08.md)
- [x] Complete S08: zero diff on double generation, custom hash preservation, published migration immutability. (Deterministic output, custom preservation, CRUD compilation, and published migration immutability contracts passed on Pages/News fixtures.)
- [ ] S09: SQLite ↔ PostgreSQL export/import; identity/revision/locale/relations are preserved, secret/session does not leak. (Allowlist-based deterministic SQLite and transient-schema PostgreSQL round-trip contracts, auth/session exclusion, large SQLite fixture with 256 records, reproducible portable round-trip benchmark, and atomic `hydro portable export|import` CLI flow with explicit confirm/dry-run support are ready; comparative benchmark with real customer hardware is open.) [R5 validation report](validation/R5-portable-export-2026-09-08.md)
- [ ] Certify Standard/PostgreSQL profile, supported OS/DB/provider versions, and upgrade path. (Linux/Windows CI support matrix initial evidence with typed profile budgets and pool implementation is ready; real PostgreSQL benchmark, provider/upgrade certification are open.) [Support matrix](SUPPORT_MATRIX.md)
- [ ] SDK v1, deprecation/support schedule, release signature, license inventory, and reproducible build. (SDK v1 candidate, same-major compatibility contract, [support policy](SDK_SUPPORT_POLICY.md), deterministic Ed25519 manifest/signature, and builder/toolchain provenance attestation contracts tied to manifest subject are ready, along with true binary reproducibility check on the same CI runner/toolchain; official deprecation dates, production key ceremony, publishing external provenance, and cross-OS reproducibility are open.) [Release validation report](validation/R5-release-manifest-2026-09-08.md)
- [ ] Measure time by comparing a standard customer job on two developers/customers using the current method.

**Output:** Stable v1 with a documented support matrix. No promises are made for unmeasured speed/topology.

## R6 — On-demand expansion

**Dependency:** Relevant v1 contracts and validated customer need.

- [ ] S05-D: multi web/worker, L2 loss, fan-out/reconnect, generation reconciliation, and bounded fallback. (Bounded process-local invalidation fan-out, slow subscriber resync/generation reconciliation, and TwoTier contract falling back to local loader on L2 failure are ready; real multi-instance/L2 provider and partition drill are open.)
- [ ] S3-compatible media/CDN, online large backup, and GC pin races; Distributed load/failure certification.
- [ ] New DB adapter, advanced search, OIDC/SAML, or custom mobile API if there's demand; each with a separate decision and test gate.
- [ ] Commerce, AI, and multi-tenant are evaluated with separate scope/ADRs.

## Deferred product ergonomics

- [ ] Provide production-oriented minimal module presets (`compact-site`, `editorial`, `commerce`,
  and `headless`) instead of treating the all-modules example configuration as a deployment
  default. Each preset must prove that excluded modules create no routes, migrations, assets, or
  workers, and must pass the SQLite/PostgreSQL module-isolation contract before release.

## Evidence and progress tracking

The necessary conditions and evidence format for gates left open in real environments are maintained in the [Production gate evidence checklist](validation/PRODUCTION-GATES.md). Latest automated reliability and security hardening evidence: [R7 validation report](validation/R7-reliability-security-hardening-2026-09-10.md).

Every piece of work is delivered as a small reviewable change. The report includes commit, environment/toolchain/DB versions, commands, expected/observed outcome, limitations, and the related roadmap item. `docs/validation/` only holds real results; planned tests are not shown as successful. In a failed experiment, an alternative is evaluated first; if an accepted decision changes, a new ADR is written.
