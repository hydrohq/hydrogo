# HydroGo

HydroGo is a modular Go CMS and JSON API platform designed for customer-owned deployments without forking the Core.

Its purpose is:

> Give customers control of their data and deployment through a secure, portable, operable CMS that does not add infrastructure they do not use.

HydroGo is not designed as an “everything CMS.” It is a stable Core with attachable modules. Disabled modules must not create routes, migrations, workers, or mandatory external services.

## Current status

The repository provides a working technical foundation and a broad contract-test suite. Pages and News modules, Site Edition flows, authentication/MFA, media, cache, backup/restore, portable export/import, release manifests, and generator experiments are implemented.

This repository is not yet declared a fully certified, commercially supported v1 across all production profiles. The remaining R3 pilot, R5 v1, and R6 Distributed evidence gates are tracked separately in the roadmap.

### Problems currently addressed

- Page and news authoring with draft, revision, preview, scheduling, and publishing flows
- Optimistic locking to prevent silent data loss between editors
- Shared application services and authorization rules for HTML and JSON API paths
- Public/private media lifecycle and safe upload metadata handling
- Common SQLite and PostgreSQL behavioral contracts
- Bounded L1 cache, generation invalidation, and single-flight miss coalescing
- MFA, session revocation, password reset, invitations, and CSRF foundations
- Audit, transactional outbox, job leasing, and idempotency infrastructure
- SQLite snapshots, PostgreSQL dump/restore, and media manifest verification
- Hash-chained external erasure log and idempotent erasure replay
- Allowlist-based portable export/import excluding secrets and session state
- Ed25519-signed release manifests and trusted public-key verification
- Deterministic generation, Client Edition boundaries, and an SDK v1 candidate

## How it works

The web server and operational commands use the same Core inside one binary:

```text
hydro serve / migrate / doctor / upgrade check
                         │
                         ▼
              HTTP handler / JSON API
                         │
                         ▼
             Application service + policy
                         │
                         ▼
             Module repository interface
                         │
              ┌──────────┴──────────┐
              ▼                     ▼
           SQLite              PostgreSQL
```

The HTML admin panel and JSON API call the same service and policy layers. Domain and application-service code does not know SQL dialect details, cache drivers, or deployment proxy details.

### Modules and Core boundaries

- `modules/pages`: pages, revisions, drafts, previews, and publishing
- `modules/news`: review, scheduling, publishing, withdrawal, and workers
- `modules/site`: navigation, settings, SEO, redirects, and form inbox
- `internal/auth`: users, sessions, MFA, reset, and invitations
- `internal/media`: uploads, visibility, pending/ready state, and storage metadata
- `internal/cache`: bounded L1, TwoTier cache, and invalidation fanout
- `internal/backup`: SQLite/PostgreSQL snapshots, manifests, restore, and activation
- `internal/release`: manifests, signatures, attestations, retained assets, and Blue/Green state
- `internal/portable`: allowlist-based data portability
- `internal/generator`: reviewable CRUD and migration drafts from collection specs

## Quick start

Requirements:

- Go 1.27.0
- No extra service is required for SQLite
- PostgreSQL 18 and `HYDRO_TEST_POSTGRES_URL` are required for PostgreSQL contract tests

Windows PowerShell:

```powershell
go build -o bin/hydro.exe ./cmd/hydro
$env:HYDRO_DATABASE_DSN = "hydro.sqlite"
./bin/hydro.exe migrate
./bin/hydro.exe serve
```

### Live development

In local development, while the proxy address remains static (`127.0.0.1:8090`), the HydroGo application restarts internally. This preserves the browser session; Go, HTML, and Templ changes are recompiled, while CSS/JS changes simply refresh the page. If a compilation error occurs, the previous working version stays running and a red development panel showing error details appears in the browser.

```powershell
go run ./tools/devserver
```

### SDK ile hızlı proje başlatma

Yeni bir Client Edition iskeleti ve frontend entegrasyon manifesti üretmek için:

```powershell
go run ./cmd/hydro project new haber-merkezi --preset=news
cd haber-merkezi
hydro project check
```

Üretilen projede `frontend/starter.json` ve `frontend/README.md`, Hydro,
shadcn, Material, Carbon, headless veya vanilla sunum katmanlarından birini
seçerek ilerlemenizi sağlar. Collection spec'leri için `hydro resource new`
her kaynak altında tip güvenli `client.ts`, React/SWR/HTMX/vanilla adapter'ları,
`form.schema.json`, OpenAPI ve CRUD yardımcılarını üretir. Bu dosyalar Core'a
çalışma zamanı bağımlılığı eklemez; kimlik doğrulama, politika ve cache
geçersizleştirme Client Edition sınırında kalır. Ayrıntılı akış için
[`docs/SDK.md`](docs/SDK.md), [`docs/PROJECT_GENERATOR.md`](docs/PROJECT_GENERATOR.md)
ve [`docs/CLI_WORKFLOW.md`](docs/CLI_WORKFLOW.md) belgelerine bakın.

The default demo database `data/demo.sqlite` is used. Provide `--database-dsn` for a different database; proxy or internal application ports can be changed with `--address` and `--app-address` respectively.

Endpoints:

- `/healthz`: process liveness only
- `/readyz`: database and migration compatibility readiness
- `/metrics`: authenticated operational counters with bounded status labels
- Versioned JSON API routes use `/api/v1/...`; existing `/api/...` paths remain compatibility aliases and advertise their successor.
- Admin/API routes: default-deny routing and authorization policies

For PostgreSQL, set `HYDRO_DATABASE_DIALECT=postgres` and use an isolated `HYDRO_DATABASE_DSN`.

Before opening a deployment change, run the local binary smoke drill on Windows:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\smoke-local.ps1
```

The drill builds an isolated binary, migrates a temporary SQLite database, starts the
server on an ephemeral loopback port, and verifies both `/healthz` and `/readyz`.
Linux CI runs the equivalent `scripts/smoke-local.sh` drill and additionally verifies
that the server exits cleanly after `SIGTERM`.

Production HTTPS settings are explicit: set `HYDRO_SECURE_COOKIES=true` and `HYDRO_ENABLE_HSTS=true` (or the equivalent `security.secure_cookies` and `security.enable_hsts` YAML fields) together with an HTTPS `HYDRO_PUBLIC_ORIGIN`. Configuration rejects HSTS without an HTTPS canonical origin or secure cookies. These settings remain disabled by default for local HTTP development.

When `HYDRO_PUBLIC_ORIGIN` is configured, HydroGo also binds HTTP requests to that canonical host and port. Forwarded host headers are not trusted; configure the reverse proxy to preserve the public `Host` header. Leave the origin empty only for local development.

Release builds can embed identity metadata with `-ldflags "-X main.buildCommit=COMMIT -X main.buildTime=UTC_TIME"`; inspect it with `hydro version --json`.

## Operational commands

```text
hydro serve
hydro migrate
hydro version --json
hydro doctor
hydro doctor --json
hydro upgrade check
hydro create-admin --email EMAIL --password-stdin
hydro backup sqlite --destination PATH [--media-dir PATH]
hydro backup postgres --destination PATH [--media-dir PATH]
hydro backup prune --root PATH --keep N --dry-run|--confirm [--pinned NAME,...]
hydro restore verify --snapshot PATH
hydro restore stage --snapshot PATH --destination PATH --sessions-revoked --egress-quarantined --old-writes-blocked
hydro restore postgres --snapshot PATH --destination PATH --sessions-revoked --egress-quarantined --old-writes-blocked
hydro restore activate-media --staging PATH --active PATH --rollback PATH
hydro portable export --path PATH
hydro portable import --path PATH --dry-run
hydro portable import --path PATH --confirm
hydro release verify --manifest release.json --public-key release.pub
```

Portable import/export is bounded to 256 MiB by default; use `--max-bytes` when an approved larger customer export is required.

`hydro release verify` does not trust the public key embedded in the manifest by itself. It requires a trusted Ed25519 public-key file distributed by the operator.

Browser-originated state-changing requests must include a CSRF token that matches the `hydro_csrf` cookie. URL-encoded forms and small multipart forms may carry the token in the `_csrf` field. Large or chunked multipart uploads must send the same token in the `X-CSRF-Token` header; this avoids parsing an unbounded upload body merely to perform CSRF validation.

Backup and restore are deliberately staging-first. SQLite and PostgreSQL backups create a new immutable snapshot directory; PostgreSQL uses `pg_dump`/`pg_restore` with a temporary password file and never puts the password in process arguments. Restore verification only checks hashes, staging never overwrites an existing destination, and media activation requires the quarantine marker plus all three operator controls. Pause application writes and background workers before taking a live snapshot.

Portable import is also explicit: `--dry-run` validates the allowlisted, secret-free document without a database write, while a real import requires `--confirm` and runs in one transaction.

Snapshot retention is explicit as well: run `hydro backup prune --dry-run` to inspect candidates, then repeat with `--confirm`. Pinned snapshot directory names are never removed.

## Tests and production gates

Daily development checks:

```powershell
go test ./...
go vet ./...
powershell -ExecutionPolicy Bypass -File .\scripts\verify-local.ps1
```

`verify-local.ps1` is the single local developer gate: it checks formatting, vet, the full test suite, a trimpath build, Client Edition import boundaries, and both external edition builds. It does not require PostgreSQL. The production-gate script remains stricter and requires a real PostgreSQL contract database.

`hydro doctor --json` provides automation-friendly preflight output for the selected profile, database, backup tools, media writability, transport security settings, binary identity, and embedded schema migration.

Dead-lettered background jobs can be reviewed without exposing their payloads:

```text
hydro jobs dead-letter list --database-dialect sqlite --database-dsn PATH --json
hydro jobs dead-letter requeue --id JOB_ID --confirm --database-dialect sqlite --database-dsn PATH
hydro jobs dead-letter purge --older-than 720h --limit 100 --confirm --database-dialect sqlite --database-dsn PATH
```

Requeue and purge are explicit operator actions. Requeue preserves the job attempt count and last error; purge is bounded by age and batch size.

Run the automated release gates after configuring a PostgreSQL contract database:

```powershell
$env:HYDRO_TEST_POSTGRES_URL = "postgres://user:password@127.0.0.1:5432/hydro_contract?sslmode=disable"
powershell -ExecutionPolicy Bypass -File .\scripts\verify-production-gates.ps1
```

On Linux or macOS, run the equivalent gate runner:

```bash
export HYDRO_TEST_POSTGRES_URL='postgres://user:password@127.0.0.1:5432/hydro_contract?sslmode=disable'
bash ./scripts/verify-production-gates.sh
```

Both runners check formatting, vet, the full test suite, race-sensitive packages, reproducible binary builds, and the portable benchmark. They fail when PostgreSQL is unavailable instead of silently treating contract tests as skipped.

Real production gates additionally require:

- Blue/Green restart and rollback with two real processes and a reverse proxy
- Restore into clean SQLite and PostgreSQL targets
- External backup and key-recovery drills
- A production signing-key ceremony
- RPO/RTO and capacity measurements using representative customer workloads

The repository does not claim a production profile or SLA until those forms of evidence exist.

## Architectural principles

- Customer work is implemented through Client Editions and extension contracts, not HydroCore forks.
- SQLite and PostgreSQL implement the same domain and application contracts.
- Migration, cache, job, asset, and release changes move forward through explicit contract tests.
- Unused services are disabled in Compact profiles; Distributed features are not defaults.
- Secure-by-default routing, default-deny authorization, and data portability are core contracts.

## Documentation

- [Roadmap](docs/ROADMAP.md): phases, open gates, and acceptance criteria
- [HydroGo philosophy](docs/HYDROGO_PHILOSOPHY.md): product and architectural principles
- [Implementation plan](docs/implementation_plan.md): scope and target architecture
- [ADR index](docs/architecture/README.md): binding architectural decisions
- [Production gate checklist](docs/validation/PRODUCTION-GATES.md): evidence format for real environments
- [Open gates and operator actions](docs/OPEN_GATES_AND_OPERATOR_ACTIONS.md): user actions, unverified tests, and production blockers
- [Operations runbook](docs/OPERATIONS_RUNBOOK.md): deploy, backup, restore, and alarm flows
- [Support matrix](docs/SUPPORT_MATRIX.md): supported and uncertified profiles
- [Compatibility matrix](docs/COMPATIBILITY_MATRIX.md): old/new release boundaries
- [Threat model](docs/security/THREAT_MODEL.md): security assumptions and open risks
- [Project status](docs/PROJECT_STATUS.md): what HydroGo currently provides and the evidence still required
- [Security policy](SECURITY.md): private vulnerability reporting and disclosure guidance
- [Contribution guide](CONTRIBUTING.md) and [repository rules](AGENTS.md)

Dependency vulnerability scanning (`govulncheck`), pinned GitHub Actions, weekly
Dependabot updates, and CodeQL Go analysis run through the repository CI. These
checks improve detection and supply-chain integrity; they do not replace the
deployment-specific production drills listed above.

## Repository layout

| Directory | Responsibility |
|---|---|
| `cmd/hydro/` | Application and operational command entry point |
| `internal/` | Core implementation details; not the external SDK |
| `extension/` | Experimental v0 customer/module contracts |
| `modules/pages/`, `modules/news/` | Module models, services, repositories, and transport |
| `migrations/` | Dialect SQL and compatibility metadata |
| `tests/` | Shared contracts, integration tests, and consumer fixtures |
| `tools/`, `scripts/` | Development tools and production-gate automation |
| `docs/validation/` | Experiment reports and verification gates |

Commercial SLA claims for Compact or Standard remain blocked until the R3/R5 evidence gates are closed. Distributed, Redis/Valkey, S3/CDN, OIDC/SAML, advanced search, commerce, AI, and multi-tenant features require a real customer need, a separate ADR, and their own verification gate.

The customer delivery and redistribution license has not yet been selected; this repository should not be treated as published under an open-source license.
