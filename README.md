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

Endpoints:

- `/healthz`: process liveness only
- `/readyz`: database and migration compatibility readiness
- `/metrics`: authenticated operational counters with bounded status labels
- Admin/API routes: default-deny routing and authorization policies

For PostgreSQL, set `HYDRO_DATABASE_DIALECT=postgres` and use an isolated `HYDRO_DATABASE_DSN`.

Production HTTPS settings are explicit: set `HYDRO_SECURE_COOKIES=true` and `HYDRO_ENABLE_HSTS=true` (or the equivalent `security.secure_cookies` and `security.enable_hsts` YAML fields). They remain disabled by default for local HTTP development.

## Operational commands

```text
hydro serve
hydro migrate
hydro doctor
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

`hydro release verify` does not trust the public key embedded in the manifest by itself. It requires a trusted Ed25519 public-key file distributed by the operator.

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

Run the automated release gates after configuring a PostgreSQL contract database:

```powershell
$env:HYDRO_TEST_POSTGRES_URL = "postgres://user:password@127.0.0.1:5432/hydro_contract?sslmode=disable"
powershell -ExecutionPolicy Bypass -File .\scripts\verify-production-gates.ps1
```

The runner checks formatting, vet, the full test suite, race-sensitive packages, reproducible binary builds, and the portable benchmark. It fails when PostgreSQL is unavailable instead of silently treating contract tests as skipped.

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
- [Operations runbook](docs/OPERATIONS_RUNBOOK.md): deploy, backup, restore, and alarm flows
- [Support matrix](docs/SUPPORT_MATRIX.md): supported and uncertified profiles
- [Compatibility matrix](docs/COMPATIBILITY_MATRIX.md): old/new release boundaries
- [Threat model](docs/security/THREAT_MODEL.md): security assumptions and open risks
- [Contribution guide](CONTRIBUTING.md) and [repository rules](AGENTS.md)

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
