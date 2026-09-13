# HydroGo (HydroCore) — Unified Implementation Plan and Production Architecture

> **Document status:** Master plan for implementation  
> **Project code name:** `HydroGo` / `HydroCore`  
> **CLI:** `hydro` (short alias `hy` optionally)  
> **First release target:** Rebrandable CMS and JSON API that supports the same business contracts on SQLite and PostgreSQL, with new databases added via certified adapters  
> **Deployment model:** Application assets embedded in a single executable; database, media, config, and secrets in an external persistent data directory

> **Realization status (2026-09-09):** R0-R2 and R4 scopes, R5 generator scope, and a significant portion of the R3/R5 validation infrastructure of the plan have been implemented and validated via contract tests. Open gates requiring a true customer/prod environment are tracked in the [roadmap](ROADMAP.md) and [production gate checklist](validation/PRODUCTION-GATES.md). This document is the primary source of scope and acceptance criteria.

## 1. Purpose and scope

HydroGo is a modular CMS product factory developed in Go for corporate sites, news portals, and mobile application backends, which does not require Node.js/npm during runtime and production build, and where the HTML admin panel and JSON API use the same business logic. The same Core contracts are used across Compact, Standard, and Distributed capacity profiles.

The primary source for architectural decisions is the [`docs/architecture`](architecture/README.md) directory. New developments must be consistent with relevant ADRs.

The initial release focuses solely on proving the following result:

- Admin can log in securely.
- Authorized editor can create a page, preview a draft revision, and publish it. News proves the review and scheduling behavior in a second module.
- HTML admin panel and JSON API enforce the same validation and authorization rules.
- Concurrent changes by two editors do not silently overwrite each other.
- Database and media can be backed up together and restored to a clean installation.
- Customer view customizations are validated at compile or startup time.

The first commercial product is the Site Edition; scope and pilot/v1/Distributed separation are in [ADR-0009](architecture/0009-site-edition-and-delivery-scope.md). Form Builder, e-commerce, and AI are subsequent scopes. SQLite provides fast local development; PostgreSQL contract tests start from the first Pages slice and are completed before the stable v1. MySQL and other engines are added with adapter certification upon real demand.

## 2. Core architectural decisions

### 2.1 Layers and dependency direction

The flow is as follows:

```text
HTTP Handler (Templ/JSON)
        ↓
Application Service
        ↓
Module-owned Repository Interface
        ↓
SQLite / PostgreSQL / Future Certified Adapter
```

- HTML and JSON handlers do not directly access the database.
- Validation, authorization, state transitions, transaction boundaries, and audit generation live in application services.
- Models, services, and repository interfaces related to news reside under `modules/news`.
- HydroCore implements shared infrastructure under `internal/`; the versioned customer/module surface is `extension/` and declared module APIs. The media domain is separate from storage infra. The Client Edition uses Core as a separate Go module; details are in [ADR-0010](architecture/0010-sdk-lifecycle-and-edition-manifest.md).
- Service methods take `context.Context`; carrying database and external operation timeouts.
- Packages do not rely on global variables; dependencies are explicitly bound at application startup.

### 2.2 Working module first, then generator

The first Pages module, followed by the News module with its workflows, are developed end-to-end manually. SDK v0 is validated with two different Client Editions; then the v1 surface and code generator are fixed. Details specific to one instance do not turn into a permanent generator contract. The standard CRUD schema is a build input; automatic DDL is not applied at runtime.

### 2.3 Single binary contract

For the single executable goal, the web server and operational commands are offered as subcommands under the same binary:

```text
hydro serve
hydro migrate
hydro backup
hydro restore
hydro admin reset-password
hydro generate module
hydro doctor
```

Templ outputs, static CSS, and static client libraries are bundled into the binary via `embed.FS`. Database, uploaded media, configuration, encryption keys, and runtime logs are not embedded; they are kept in external persistent resources. Hashed assets needed by previous HTML can be extracted from the signed inventory to a common asset directory during deployment and retained for their retention period. Although the application binary includes maintenance tools, the generator is not considered complete until the first stable SDK is validated.

## 3. Folder structure

This is a target layout; currently, the repository only has docs. Separate packages are not immediately created for every responsibility inside a module; they are split when the need arises. Instead of the old same-module example under `client/`, an external Client Edition is used.

```text
C:\dev\cms
├── cmd/
│   └── hydro/                 # Single entry point and subcommands
├── internal/
│   ├── audit/                 # Application-level append-only audit
│   ├── auth/                  # User, password, session, and authorization infra
│   ├── config/                # Typed configuration and startup validation
│   ├── database/              # SQLite/PostgreSQL and extensible adapters
│   ├── cache/                 # Noop, bounded memory, and Redis/Valkey cache
│   ├── jobs/                  # Durable jobs, retry, and email worker
│   ├── mailer/                # Email provider interface
│   ├── media/                 # File validation and storage
│   ├── observability/         # Log, metric, and health/readiness
│   ├── router/                # Secure router and middleware chain
│   └── ui/                    # Internal layout/Templ implementation
├── extension/                 # Experimental v0; versioned SDK v1 post-proof
├── modules/
│   ├── pages/                 # First end-to-end content module
│   └── news/
│       ├── domain/            # Article and state rules
│       ├── service/           # Use cases and authorization
│       ├── repository/        # Interface and certified dialect implementations
│       ├── httpadmin/         # Templ/HTMX handlers
│       ├── httpapi/           # JSON handlers
│       └── ui/                # News Templ components
├── examples/
│   └── client-edition/        # External consumer fixture via separate Go module
├── migrations/                # Dialect-based, sequential, and immutable migrations
├── docs/
│   └── architecture/          # ADR records and decision index
├── static/
│   ├── vendor/                # Pinned htmx, Alpine CSP, Quill files
│   └── css/                   # Pre-generated Tailwind output
├── tests/
│   ├── integration/
│   └── fixtures/
├── tools/                     # Version definitions of development tools
├── Makefile
├── go.mod
├── go.sum
└── config.env.example
```

`schema/user.yaml` will not be used. Runtime schema is managed by SQL migrations. The YAML definition that the generator will take in the future is kept in a separate `module-specs/` directory and is not mixed with the database schema.

A real customer project has a separate repository with its own `edition.yaml`, `edition.lock`, `custom/`, `generated/`, and build entry. Signed release metadata is produced with `release.json`. File formats are pinned during the ADR-0010 validation phase.

## 4. Security architecture

Identity lifecycle is in [ADR-0013](architecture/0013-identity-and-security-boundaries.md), actors/trust boundaries and attack tests are in the [threat model](security/THREAT_MODEL.md). Trusted module code is not sandboxed within the same process. In the commercial pilot, internet-accessible admin/publisher accounts must pass an MFA gate.

### 4.1 Default deny access

- Routes are explicitly registered as public or auth-required routes; an unclassified route initially yields an error.
- Authorization does not rely solely on roles. Record and state-based policies like `CanEdit`, `CanSubmit`, `CanPublish`, `CanDelete` are used.
- API and HTML flows call the same service methods.
- Authentication failure and authorization denial yield proper `401/403` behavior; `404` is used where exposing the existence of a sensitive record is unwanted.

### 4.2 Session and password security

- Passwords are stored with Argon2id using versioned parameters; parameters are determined by security testing and server capacity.
- Session IDs are generated with cryptographic randomness; a hash is stored in the database instead of the raw value.
- Cookies will be `HttpOnly`, `Secure`, and `SameSite=Lax`; HTTPS is mandatory in production.
- Session ID is refreshed on login. Absolute session duration and idle timeout are defined separately.
- Password change and emergency CLI reset revoke all active sessions of the user.
- User and IP-level rate limiting with increasing backoff is applied at the login endpoint.
- Re-authentication may be required in sensitive forms.
- The initial admin is not automatically created with a random startup password; it is created via an explicit bootstrap command.

### 4.3 CSRF, CSP, and client-side security

- Bound CSRF tokens are validated on state-changing HTML requests.
- Strict CSP without requiring nonces/hashes is applied with `script-src 'self'`; inline scripts and event handlers are not used.
- Alpine.js CSP distribution is used.
- `allowEval: false` and `allowScriptTags: false` are set for HTMX.
- `frame-ancestors 'none'`, `base-uri 'self'`, `object-src 'none'`, proper `Referrer-Policy`, `X-Content-Type-Options`, and `Permissions-Policy` are sent.
- Security headers and CSP violations are tested automatically.

### 4.4 Rich text security

- The first editor candidate is Quill. The Delta JSON is stored inside a `format`, `format_version`, `document` envelope; clean HTML is a derived output carrying the renderer/sanitizer version. Reading the old format and loss reporting on conversion are defined in [ADR-0012](architecture/0012-content-media-and-portability.md).
- The server only accepts defined Delta operations; HTML coming from the client is not considered trusted.
- HTML is sanitized on the server with an allowlist. Besides tags, attributes and URL protocols are also restricted.
- `javascript:`, dangerous `data:` types, event handlers, `style`, `iframe`, `script`, and unknown attributes are rejected.
- The external image policy is explicitly defined; only local media URLs are allowed in the first version.
- The sanitizer version is stored with the content. When the policy changes, a command to bulk-regenerate HTML from Delta is provided.
- `templ.Raw` can only be used on typed safe content produced by the sanitizer.

### 4.5 Media upload

- File extension, declared MIME, and file signature are checked together.
- Allowed formats are restricted to JPEG, PNG, and WebP in the first release; SVG is not accepted.
- Byte, pixel, and dimension limits are applied for the file and decoded image.
- Images are decoded and re-encoded to strip dangerous payload and metadata.
- The client filename is not used as a storage key; a random/content-hash based name is used.
- Media is stored outside the web root and served with the correct `Content-Type` via a controlled handler.
- Orphan media cleanup runs as a safe job based on references and retention period.
- Private/public visibility, pending→ready upload, and crash/GC behavior are subject to ADR-0012. Draft media is not considered public just because it has a hashed URL; backup pins and revision references are protected from GC.

## 5. Database, cache, and consistency

### 5.1 Database profiles

Domain and application service code do not know the SQL dialect. In the first release, SQLite and PostgreSQL implement the same module repository contracts; queries, migrations, and transaction details belong to the driver. New engines are added as certified adapters with capability and contract tests. Detailed decisions are defined in [ADR-0006](architecture/0006-extensible-database-adapters.md).

#### SQLite

The first release runs on a single application instance and local persistent NVMe/SSD. NFS and network file systems are not supported.

The following settings are applied on every new connection and their results are verified at startup:

```sql
PRAGMA journal_mode = WAL;
PRAGMA busy_timeout = 5000;
PRAGMA synchronous = NORMAL;
PRAGMA foreign_keys = ON;
```

`synchronous=NORMAL` is considered a balance of performance and safety; it is logged as an operational decision that the last committed transactions might be lost on power loss. `FULL` or PostgreSQL is used for financial/order profiles.

- Connection pool size is limited according to SQLite's single-writer nature.
- Write transactions are kept short; network, email, or file conversion operations are not done inside a transaction.
- Migrations are versioned, sequential, and immutable once published.
- The application validates the required migration/checksum and the declared schema read/write compatibility per Core/module. After a compatible expand migration, an old binary can start; schemas outside the support range are rejected. Details are in [ADR-0011](architecture/0011-consistency-upgrade-and-recovery.md).
- An automatic backup and rollback procedure prior to migration is defined.
- `PRAGMA integrity_check` and `foreign_key_check` are run during maintenance/restore validation.

#### PostgreSQL

PostgreSQL is the recommended driver for multi-instance, high write concurrency, and scale-out setups. Connection pool, isolation level, serialization/deadlock retry, and online migration behavior are explicitly configured.

#### Future adapters

MySQL and other databases are not considered supported just by adding a connection driver. They become certified adapters when module queries, migration, error mapping, transaction/locking, backup/restore, and common contract tests are completed.

### 5.2 Transaction boundaries and durable jobs

A content change, new revision, valid state transition, audit event, and (if needed) durable job record are written within a single transaction. Repository, audit, and outbox are bound from the same transaction scope; they are not split into independent pool calls. External effects like emails and future webhooks are processed from the durable job queue after commit.

- Jobs carry unique idempotency keys.
- Bounded exponential backoff and dead-letter states are used on errors. For the News worker, retry delay is bounded; persistent/poison payloads are dead-lettered with lease validation and the last error diagnosis is stored.
- When the worker restarts, half-finished jobs safely resume.
- For repeated commands, in-DB publish changes are kept idempotent. External effects are delivered at-least-once; when the provider offers no idempotency, email timeouts carry ambiguous result/retry risks. General exactly-once guarantees are not provided. Lease, fencing, consumer state, and payload compatibility are defined in ADR-0011.

### 5.3 Concurrent editing

An incrementing `version` field exists in the `articles` table. Updates are performed with this condition:

```sql
UPDATE articles
SET ..., version = version + 1
WHERE id = ? AND version = ?;
```

If zero rows change, the service returns a `conflict` result. The admin panel gives the user the ability to reload the record and compare changes; a silent last-writer-wins behavior is not used.

### 5.4 Multi-tier cache

The cache consists of request-scope memoization, bounded L1 memory, optional Redis/Valkey-compatible L2, and HTTP/CDN L3 tiers. An external cache service is not required in the Compact profile; distributed cache is mandatory in the Distributed profile.

- The cache only carries reproducible data; bounded DB fallback and backpressure are applied on failure. Controlled 429/503 may occur on capacity overflow; the fact that core data isn't lost does not mean a zero-downtime guarantee.
- TTL, jitter, versioned keys, tag invalidation, stampede protection, and size limits are mandatory.
- A publish transaction writes an outbox event for cache/CDN invalidation.
- Admin, preview, session, audit, money, and stock records do not enter the public/shared cache.
- Hit/miss, latency, eviction, byte usage, and invalidation failures are measured.

Detailed tier contracts are in [ADR-0003](architecture/0003-cache-architecture.md); current fan-out/generation/stale budget and failure rules are in [ADR-0011](architecture/0011-consistency-upgrade-and-recovery.md). The worker's local L1 flush does not flush other web processes' L1; split topology is not certified until it passes this test.

## 6. News domain model

This section is the behavior of the second module, News; the first Pages slice is built on draft/publish/revision. Common content ID, locale, route, and media boundaries are in [ADR-0012](architecture/0012-content-media-and-portability.md). The entire workflow of News is not imposed on the general Pages model.

### 6.1 States

```text
draft → in_review → scheduled → published → archived
   ↑         ↓           ↓           ↓
   └─────────┴───────────┴───────────┘ (authorized withdrawal)
```

- `preview` is not a persistent state; it is the action of an authorized user viewing a specific revision.
- Source state, target state, and required authorization of every transition are defined in an explicit table.
- Scheduled publishing is stored as UTC; translated to the user's timezone in the UI.
- Editing published content creates a new draft revision; the live version does not change at the moment of saving.
- Preview URLs are short-lived, signed, tied to a revision, and are not indexed by search engines.

### 6.2 Identity, slug, and SEO

- Record ID is a stable ID independent of the URL.
- Slug uniqueness and normalization rules are defined.
- Changing a draft slug does not affect the live path. If the previous active path is needed during publish, it is added to a 301 redirect; path conflict and cycle checks are applied in the same transaction. [ADR-0014](architecture/0014-publication-routes-and-erasure-recovery.md) is binding.
- Canonical URL, meta title/description, Open Graph fields, sitemap, and historical URL management are Pilot Site Edition acceptance criteria (Phase 3). The first Pages slice proves route identity and draft/publish atomicity.
- Deletion is a soft-delete initially; hard deletion requires separate authorization, waiting period, and audit.

## 7. API contract

- The API is versioned under `/api/v1`. Initial `/api/...` paths are preserved as backward compatibility aliases and route to the transition using `Deprecation`/`Link` headers.
- List endpoints use cursor-based pagination and upper limits.
- Errors return in a single machine-readable format: `code`, `message`, `fields`, `request_id`.
- Dates are sent as RFC 3339 in UTC.
- Unknown fields in request bodies are rejected; size limits are applied.
- Only permitted fields are writable; role, publish state, or ownership cannot be changed via mass model binding.
- Public APIs return only the published, current revision. Drafts and scheduled content do not leak in unauthorized calls.
- Optimistic locking is also applied in API updates using ETag/`If-Match` or an explicit version field.
- The OpenAPI contract is versioned alongside the code and validated in contract tests.
- Mobile client authentication is not published without a separate ADR determining the token/OAuth model distinct from the admin panel session.

## 8. View Registry and customer customization

It is a fixed-contract typed view/component slot; the runtime registry is used if these behaviors are truly needed and proven in two themes. Initial implementation may start with compile-time composition. SDK maturity is managed with [ADR-0010](architecture/0010-sdk-lifecycle-and-edition-manifest.md).

- Core and modules explicitly register stable view keys.
- If a customer override uses an unknown key, missing parameter, or incompatible contract, compilation or startup fails.
- Templ components are compile-time Go code; arbitrary `.templ` files are not loaded at runtime.
- Overrides are compiled into the same binary. Changing a customer design requires a new build.
- Registry conflicts are not silently resolved via non-deterministic loading; double registrations generate errors.
- News card and detail page overrides are validated by snapshot/golden visual or HTML structure tests.

## 9. Backup and restore

Current protocol is in [ADR-0011](architecture/0011-consistency-upgrade-and-recovery.md). Backup, logical export, and binary rollback are separate operations.

### 9.1 Consistent backup

`hydro backup` uses the driver's reliable snapshot mechanism. SQLite Online Backup API or verified `VACUUM INTO`; PostgreSQL uses a version-appropriate backup tool or certified provider snapshot/PITR integration. Adapter tool/version requirements are validated by `hydro doctor`. Having a single application binary does not mean external DB backup tools are unnecessary.

- The first Compact method can perform a controlled halt on application/worker writes. Byte and duration budgets are pre-measured; there is no guarantee for a brief pause for the entire media copy.
- In the large/distributed method, the referenced immutable media manifest of the DB snapshot is protected by a GC pin/retention. Race between GC and pin is prevented; not all writes are paused during the large copy. This method is not offered before passing the profile test.
- The manifest includes the backup ID, Core/module/schema versions, file/object inventory, and SHA-256 verifications. A backup with missing media is not considered complete.
- The archive uses documented strong encryption; the key is not stored in plaintext at the same location. Secret/key recovery is via a separate controlled channel.
- Temporary output is not marked complete until verified. Failures safely release locks/pins, without prematurely deleting objects that must be kept.
- Off-server copy, retention, last success/age alarm, and regular restore drills are mandatory.

### 9.2 Restore

`hydro restore` does not directly overwrite the running application's target; it uses an empty/staging target. First, manifest, version, DB integrity/references, and media checksums are verified.

- SQLite is prepared in an offline staging data directory and deployed appropriately for the platform; WAL/SHM and open processes are checked additionally.
- PostgreSQL is restored to a separate DB/instance. After smoke validation and a controlled write halt, the connection target is swapped; directory renaming is not a PostgreSQL restore.
- Workers and external egress are disabled on first boot. Old outbox/email/webhook effects are reconciled; sessions are revoked by default. Per ADR-0014, deletion history is fetched from an external log independent of the restore; traffic is not opened until it is verified as current and complete. Secret/key versions are evaluated.
- The old deployment is fenced off so it cannot write again. Binary rollback is not mixed with restoring from backup; restoring can lose writes after the backup.

RPO/RTO, acceptable write pause, retention, and key recovery targets are quantified before the first pilot. Without verifying DB PITR and media history together, a whole-product PITR guarantee is not provided. DB independent export/import model is subject to ADR-0012.

## 10. Audit, privacy, and data lifecycle

- Login, failed login, password/session change, content creation/update/state transition/deletion, authorization change, backup, and restore are audited.
- An audit event contains the actor, action, target, time, request ID, and a safe summary; request ID is carried from the HTTP context in `audit_events.request_id`. It does not contain passwords, session tokens, or sensitive full bodies.
- The Audit API is append-only. PostgreSQL runtime role cannot run UPDATE/DELETE; the migration/maintenance role is separate. There is no equivalent DB role guarantee against someone with full access to the SQLite file; OS isolation and an external audit target if needed are used. There is a retention/export policy; details are in ADR-0013.
- For GDPR/KVKK, what personal data is processed, why, retention period, deletion/anonymization, and data export processes are defined.
- A consent banner is only shown if there is client storage/tracking that truly requires consent; a mandatory session cookie is separated from analytics consent.

## 11. Observability and operations

- `hydro serve` writes request ID, method, path, status, and duration in JSON-formatted structured HTTP logs; query strings, body, cookies, and response body are not logged. Secrets and personal content are masked.
- `/healthz` only shows process liveness, `/readyz` shows database and migration compatibility.
- HTTP server includes read-header, read, write, and idle timeouts; body size limits, and controlled graceful shutdown.
- Key metrics: request duration/error rate, readiness state/failure, DB pool/lock/deadlock/busy status, cache hit/miss/eviction, queue depth, failed jobs, disk usage, backup age, and last backup result. Process-local HTTP request/status/latency, `hydro_readiness_state` (instant 0/1), `hydro_readiness_failures_total` (cumulative event count), `sql.DB.Stats()` pool, public cache hit/miss/load/error/eviction/expiration, durable jobs depth/leased and worker processed/failure/enqueue-error counters are served on the authenticated `/metrics` endpoint with a bounded label set; DB lock/deadlock, queue retry/failure, disk/exporter integration are separate ops gates.
- Alarms are defined for disk full threshold, certificate expiration, failed backup, and migration failure.
- Build version, commit, and schema compatibility are available in `hydro version` and diagnostics output protected by access; public health/readiness does not return detailed internal info.
- Secrets are sourced via environment or a secret file; they are not logged, and real values are not placed in `config.env.example`.

## 12. Developer experience and supply chain

- Node.js/npm is not required to compile the production binary. Tailwind CSS output is generated with a pinned standalone tool during development or release prep, and checked into the repo.
- Go, Templ, Air, Tailwind standalone, and other tool versions are pinned.
- PowerShell commands for Windows or a cross-platform Go-based task tool are provided; `make` will not be the sole mandatory path.
- Third-party JS files are pinned with version and SHA-256; their licenses are included in the distribution.
- Release build passes reproducibility, `go test`, `go vet`, formatting, migration, and embedded asset checks.
- CI verifies whether `go:generate` outputs are up to date.

## 13. Test strategy

Tests prove layer behavior and critical risks:

- **Unit:** State transitions, authorization policies, slug, and sanitizer rules.
- **Repository integration:** Common contract tests on SQLite and PostgreSQL; migration, foreign keys, transaction, deadlock/retry, and optimistic locking. Future adapters use the same test kit.
- **HTTP integration:** CSRF/CSP, auth, HTML/API behavior parity, proper error codes, and absence of draft leaks.
- **Security:** Malicious HTML/URL samples, file signature mismatch, overly large image, session fixation, and rate limit.
- **Cache:** TTL/jitter, invalidation, stampede, bounded memory, safe fallback on L2 failure, and no caching of sensitive content.
- **Backup/restore:** Consistent snapshot under concurrent writes in each DB driver, manifest corruption, missing media, and real restore to a clean installation.
- **Capacity profiles:** Compact resource budgets and Distributed concurrency/topology tests.
- **Migration:** Transition from supported legacy production schema, old binary restart post-compatible expand, out-of-bounds schema rejection, and online/backfill/maintenance behavior for each dialect.
- **Concurrency:** When two editors save the same version, one receives an explicit conflict.
- **View Registry:** Fail-fast behavior for unknown overrides and double registrations.
- **Recovery/deployment:** Old binary rollback after a new write; retained assets; restored session revocation and job/egress quarantine; not running contract tests that write to a real customer DB.

Tests are not merely superficial checks that repeat the implementation; each proves a defined risk.

## 14. Phased roadmap and exit criteria

Every phase closes with implementation/test evidence. An accepted ADR does not mean the implementation is complete. Detailed experiments are tracked in the [pre-implementation review](PRE_IMPLEMENTATION_REVIEW.md).

Job-by-job progress and evidence gates are tracked in the [Roadmap](ROADMAP.md); R0–R6 align with these phases.

### Phase 0 — Architectural foundation and risk log

- First product/tenancy, schema/custom boundary, SDK lifecycle, cache/transaction, migration, content/media, and auth decisions are clarified with ADR-0009–0013.
- Threat model, out-of-scope features, and the validation method for decisions are recorded.
- It is specified which open questions block which deliveries; there is no requirement to lock down all future details.

**Output:** Core decisions are consistent; the owner, experiment, and gate of remaining uncertainties are clear. This document update provides the design part, technical validation waits in the following phase.

### Phase 1 — Technical experiments and walking skeleton

- External Client Edition import boundary, Go module path, toolchain/driver/migration/UI dependencies are evaluated and versioned.
- `go mod`, single `hydro` entry point, typed config, manifest preflight, health/readiness, and graceful shutdown.
- SQLite and PostgreSQL connection, transaction-scope, migration ledger/compatibility experiment.
- Default-deny router, session/bootstrap, CSRF/CSP, and basic audit.
- Compact resource budgets, bounded L1 and public HTTP cache; extra queue/cache services not mandatory.
- Windows development and Linux amd64 build flow; Templ/editor/HTMX CSP compliance.

**Output:** Build/entry succeeds in a clean environment; transaction rollback and required migration checks pass in both DBs. Concrete technology choices are recorded with experiment results.

### Phase 2 — Manual Pages and first Client Edition

- Pages create/edit, draft/revision/publish, optimistic locking, locale/route fundamentals.
- Admin, public HTML, and JSON run through the same service/policy.
- Versioned rich-text, sanitizer, and private/public media flow.
- Local storage, limited cache invalidation, and persistent outbox.
- Real customer theme with typed view slots; no Core fork or internal imports.

**Output:** Two-editor conflict, draft/private media leak, and SQLite/PostgreSQL common behavior tests pass. SDK is still v0.

### Phase 3 — Pilot Site Edition and operations

- Navigation, settings, SEO/redirects, limited form inbox, revisions, and public API are completed.
- MFA, invite/reset/recover, role/session revoke; security fixtures.
- Backup/restore, external copy, key recovery; restore job/egress quarantine, and external erasure log/checkpoint experiment (ADR-0014).
- Measured RPO/RTO, resource/soak tests, log/metric/alarms, data retention, and runbook.
- Online migration, blue/green, legacy asset, and rollback experiment on the single server profile to be certified.

**Output:** The real customer flow, security, and operation gates of the pilot pass. No unverified Distributed support or stable SDK promise is made.

### Phase 4 — News, second Edition, and SDK validation

- News: draft/in-review/scheduled/published/archived, record/field authorization, schedule cancellation, revision preview.
- Second theme and custom business rule; same transaction/extension boundary across two modules.
- Job lease, retry/reorder, migration, and cache contract kits for two DBs.
- Core/module/config/content/event/job/asset release compatibility matrix; backward compatibility fixtures.

**Output:** Two different modules and two branded external Client Editions pass; the small surface to be opened for SDK v1 is determined. The future DB certification kit is ready for release.

### Phase 5 — Generator and stable v1

- Go/Templ CRUD and reviewable dialect migration draft from versioned collection spec.
- Deterministic generation, diff, generated/custom protection; published migration checksums are immutable.
- Authorization defaults to deny; custom business rules and destructive changes are not guessed.
- Duration measurement on a real standard customer job; including regeneration and custom rule change.
- SDK v1, SQLite/PostgreSQL certification, portable export/import fixture, supported topology/OS/DB version matrix.
- License/support/deprecation schedule and upgrade guide are finalized before publication.

**Output:** Salable v1 with documented scope; unmeasured profile metrics are not passed off as success.

### Phase 6 — Expansion with demand

- Distributed profile with multi web/worker, L2, S3-compatible object storage, and CDN on PostgreSQL; fan-out/reconnect, overload, and failure tests.
- MySQL or a new certified adapter based on real demand; bulk data transfer.
- Form Builder, advanced search, OIDC/SAML, and private mobile API with separate scope/threat reviews.
- Commerce money/stock/transaction/ledger via ADRs; AI designed separately with authorization, privacy, audit, cost, and human oversight.

**Output:** Each new feature and profile is added to the support matrix with its own evidence.

## 15. Production release checklist

- [ ] All routes classified as public/authorized and pass default-deny test.
- [ ] Session refresh, timeout, revocation, and rate limit tests pass.
- [ ] Admin/publisher MFA, invite/reset, and recovery flows validated for the commercial pilot.
- [ ] HTML sanitizer and media abuse tests pass.
- [ ] SQLite PRAGMAs and PostgreSQL pool/isolation settings are validated at startup.
- [ ] Passes common contract and migration tests for SQLite and PostgreSQL.
- [ ] Cache is bounded, observable, and falls back safely on L2 failure.
- [ ] Benchmark/soak tests pass under defined reference load for every profile declared supported by the release; Distributed is additionally certified.
- [ ] Releases with online migrations do not cause public downtime during blue/green.
- [ ] Concurrent editing does not produce silent data loss.
- [ ] Public API does not leak draft, deleted, or scheduled content.
- [ ] Migration tested on an old production copy.
- [ ] Backup stored off-server and restored to a clean environment.
- [ ] Post-restore session revocation and job/egress quarantine validated; past deletions reconciled.
- [ ] RPO/RTO, retention, and key recovery procedure written.
- [ ] Verified no passwords, tokens, or sensitive content in logs.
- [ ] HTTP timeout, graceful shutdown, disk, and backup alarms working.
- [ ] Binary, JS/CSS assets, licenses, and version info are complete.
- [ ] Release signature/manifest validated; schema/content/job/config compatibility and previous asset retention tested.
- [ ] Runbook covers deploy, rollback, backup, restore, password reset, and disk-full scenarios.

## 16. Deferred decisions

The boundary of support and deferral for the following topics is clear. Relevant ADRs will be updated or new decisions written before new capabilities are unblocked:

- Running the SQLite file on NFS or object storage is not supported. An S3-compatible adapter for media is planned via ADR-0012; it will be implemented before Distributed certification.
- Multi-tenant data model.
- Mobile-specific API token/OAuth model.
- E-commerce payment, stock, and order consistency.
- Seamless live data transfer between certified databases.
- Write/publish authorization for AI agents.

The success of this plan is not measured by the creation of files and layers, but by the end-to-end proof of secure publishing flows, conflict-free editing, and verified restores.

## 17. Hardening execution roadmap (2026-09)

The remaining improvement work is executed in small, independently verifiable
stages. Each stage must leave the repository green, update the affected ADR or
validation record, and be committed and pushed before the next stage starts.

1. **Resource and data bounds (implemented):** add pagination/maximum limits to public and
   administrative list projections, query timeouts, and regression tests for
   bounded memory use.
2. **API contracts (partially implemented):** standardize JSON errors, pagination metadata, idempotency
   behavior, OpenAPI coverage, and compatibility tests across REST/GraphQL.
3. **Commerce reconciliation (partially implemented):** expose pending payment/refund reconciliation,
   add provider replay/concurrency tests, and document operator runbooks.
4. **Module isolation:** remove inactive-module navigation and construction,
   verify route/migration/worker/resource absence, and add compact-profile tests.
5. **Identity and privacy:** finish session/device controls, MFA lifecycle,
   privacy export/erasure verification, and sensitive-log regression tests.
6. **Observability and operations:** add durable metrics/export integration,
   correlation IDs, alertable health signals, and tested backup/restore drills.
7. **Frontend maintainability:** extract large inline templates, remove browser
   persistence of member data, and complete accessible loading/error/empty states.
8. **Release certification:** run SQLite/PostgreSQL contract, load/soak, fuzz,
   E2E, migration compatibility, container hardening, and production gate checks.

Items that require provider credentials, a customer deployment, or external
infrastructure remain explicitly gated; they are not marked complete by local
unit tests alone.
