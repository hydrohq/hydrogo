# HydroGo

HydroGo is a modular Go CMS and JSON API platform designed for customer-owned deployments without forking the Core.

Its purpose is:

> Give customers control of their data and deployment through a secure, portable, operable CMS that does not add infrastructure they do not use.

HydroGo is not designed as an “everything CMS.” It is a stable Core with attachable modules. Disabled modules must not create routes, migrations, workers, or mandatory external services.

## Current status

The repository provides a working technical foundation and a broad contract-test suite. Pages and News modules, Site Edition flows, authentication/MFA, media, cache, backup/restore, portable export/import, release manifests, and generator experiments are implemented.

This repository is not yet declared a fully certified, commercially supported v1 across all production profiles. The remaining R3 pilot, R5 v1, and R6 Distributed evidence gates are tracked separately in the roadmap.

### Binary-only customer distribution

Customer installations can be delivered without Core or module source code. The vendor builds an allowlisted, stripped (optionally obfuscated) `hydro` runtime and ships only the binary, `hydro.lic`, `hydrogo.lock`, configuration, themes, and Client Edition files. The package builder rejects `internal/`, `modules/`, `cmd/`, `.git/`, `go.mod`, `go.sum`, Go source files, and private keys. See [`PROPRIETARY_NOTICE.md`](PROPRIETARY_NOTICE.md), [`docs/COMMERCIAL_LICENSING.md`](docs/COMMERCIAL_LICENSING.md), and [`docs/architecture/0143-binary-only-customer-distribution.md`](docs/architecture/0143-binary-only-customer-distribution.md).

Vendor-only signing is performed with `hydro-issuer`; the customer runtime exposes only license verification/status. Build a customer package with `scripts/build-customer-runtime.ps1` or `.sh`. `HYDRO_TRUSTED_PUBLIC_KEYS` is injected in the protected vendor build, while private signing keys never enter the customer artifact. A privileged operator can still reverse engineer a running binary; this boundary prevents source delivery and casual reuse, not every possible binary patch.

### Problems currently addressed

- Page and news authoring with draft, revision, preview, scheduling, and publishing flows
- Optimistic locking to prevent silent data loss between editors
- Shared application services and authorization rules for HTML and JSON API paths
- Public/private media lifecycle and safe upload metadata handling
- Protected private-media delivery with bounded presigned URLs for authorized editors/admins
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

### Capacity profiles and optional services

- `compact`: a development/small-installation profile using SQLite and local media without extra services.
- `standard`: the recommended profile for PostgreSQL, Redis/Valkey, and higher connection/worker budgets.
- `distributed`: a contract profile for multiple application/worker instances, Redis/Valkey L2 cache, and S3-compatible media/CDN topology; separate production certification is required.

Redis/Valkey (`HYDRO_REDIS_URL`), S3-compatible media (`HYDRO_MEDIA_STORAGE_DRIVER=s3`
and `HYDRO_MEDIA_S3_*`), CDN (`HYDRO_MEDIA_CDN_BASE_URL`), SMTP (`HYDRO_SMTP_*`), and
FFmpeg (`HYDRO_MEDIA_FFMPEG`) are used only when explicitly configured. These services
are not automatically required in Compact. Do not put provider credentials in YAML or
source code; use a secret manager or protected environment variables.

### Module and Core boundaries

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

To use the binary as `hydro`, add the `bin` directory to PATH. Without changing
PATH, prefix examples with `./bin/hydro.exe` (or `./bin/hydro` on Linux/macOS).
When no configuration file is supplied, values are read from environment
variables and then defaults; select a YAML file with `--config`.

To add it to PATH for the current PowerShell session:

```powershell
$env:Path = "$(Resolve-Path .\bin);$env:Path"
hydro version
```

Linux/macOS:

```bash
go build -o bin/hydro ./cmd/hydro
export HYDRO_DATABASE_DSN=hydro.sqlite
./bin/hydro migrate
./bin/hydro serve
```

### Live development

During local development, the proxy address (`127.0.0.1:8090`) remains stable
while the HydroGo application restarts internally. Browser sessions are preserved;
Go, HTML, and Templ changes are recompiled, while CSS/JS changes refresh the page.
If compilation fails, the previous working version stays up and a red development
panel with diagnostic details appears in the browser.

```powershell
./bin/hydro.exe run
```

`hydro run` starts the internal `tools/devserver` proxy; do not invoke that Go
path directly. `hydro dev` is a backwards-compatible alias accepting the same
options. The default proxy is `127.0.0.1:8090` and the application address is
`127.0.0.1:8091`.

If `hydro help` does not list `hydro run`, the shell is resolving an older
binary. From the repository root, rebuild it and verify the resolved executable:

```powershell
go build -trimpath -buildvcs=false -o hydro.exe ./cmd/hydro
where.exe hydro
hydro help
hydro run
```

Alternatively, bypass PATH resolution with `./bin/hydro.exe run`.

### Start a project quickly with the SDK

To generate a Client Edition scaffold and frontend integration manifest:

```powershell
./bin/hydro.exe project new haber-merkezi --preset=news
Set-Location haber-merkezi
../bin/hydro.exe project check
```

`hydro project check` also validates all resource scaffolds under `generated/`
in the same dependency-free preflight.

The generated project's `frontend/starter.json` and `frontend/README.md` let you
choose Hydro, shadcn, Material, Carbon, headless, vanilla, or custom presentation
layers. For collection specs, `hydro resource new` generates a type-safe `client.ts`,
React/SWR/HTMX/vanilla adapters, `form.schema.json`, OpenAPI, and CRUD helpers for
each resource. These files add no runtime dependency on Core; authentication,
policy, and cache invalidation remain at the Client Edition boundary. See
[`docs/SDK_CAPABILITY_MATRIX.md`](docs/SDK_CAPABILITY_MATRIX.md), [`docs/PROJECT_GENERATOR.md`](docs/PROJECT_GENERATOR.md),
and [`docs/CLI_WORKFLOW.md`](docs/CLI_WORKFLOW.md) for the detailed workflow.

Each generated collection carries its deterministic `spec.json` source;
`hydro resource check --root .` validates all resources in one command without a
database connection. Pass `--spec` to check an individual resource as well.

The recommended SDK spelling is `hydro project new`; the legacy `hydro new`
spelling remains for backwards compatibility. The generator does not copy Core
source code into the generated project; it creates only a Client Edition scaffold
and reviewable generated artifacts.

The default demo database is `data/demo.sqlite`. Use `--database-dsn` for another
database; `--address` and `--app-address` select the proxy and internal application
ports respectively.

Endpoints:

- `/healthz`: process liveness only; it does not validate dependencies
- `/readyz`: database and migration-compatibility readiness
- `/metrics`: authenticated operational counters with bounded status labels
- Versioned JSON API routes use `/api/v1/...`; existing `/api/...` paths remain compatibility aliases and advertise their successor
- Admin/API routes: default-deny routing and authorization policies

For PostgreSQL, set `HYDRO_DATABASE_DIALECT=postgres` and use an isolated `HYDRO_DATABASE_DSN`.

Before opening a deployment change, run the local binary smoke drill on Windows:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\smoke-local.ps1
```

The drill builds an isolated binary, migrates a temporary SQLite database, starts
the server on an ephemeral loopback port, and verifies `/healthz` and `/readyz`.
Linux CI runs the equivalent `scripts/smoke-local.sh` drill and additionally
verifies clean exit after `SIGTERM`.

Production HTTPS settings are explicit: set `HYDRO_SECURE_COOKIES=true` and
`HYDRO_ENABLE_HSTS=true` (or the equivalent `security.secure_cookies` and
`security.enable_hsts` YAML fields) together with an HTTPS `HYDRO_PUBLIC_ORIGIN`.
Configuration rejects HSTS without an HTTPS canonical origin or secure cookies;
these settings remain disabled by default for local HTTP development.

When `HYDRO_PUBLIC_ORIGIN` is configured, HydroGo binds HTTP requests to that
canonical host and port. Forwarded host headers are not trusted; configure the
reverse proxy to preserve the public `Host` header. Leave the origin empty only
for local development.

Release derlemelerine `-ldflags "-X main.buildCommit=COMMIT -X main.buildTime=UTC_TIME"`
with `-ldflags`; inspect it with `hydro version --json`.

Node.js/npm is not required for the runtime or production binary. When frontend
assets must be generated, pinned development/release tools are used and the
resulting outputs are verified with the binary.

## Common CLI commands

```text
hydro run [--address ADDR]
hydro new PROJECT [--preset PRESET]  (legacy alias; recommended: hydro project new)
hydro serve
hydro migrate
hydro version --json
hydro doctor
hydro doctor --json
hydro upgrade check
hydro create-admin --email EMAIL --password-stdin
hydro seed-demo --confirm-demo
hydro validate --config config.yaml
hydro test --package ./modules/news/... --timeout 2m
hydro generate module --spec module.yaml --out modules/catalog
hydro module create --spec module.yaml --out modules/catalog
hydro resource new --name article --field title:text --out .
hydro frontend check --manifest frontend/starter.json
hydro api diff --baseline openapi.json --current openapi.json
hydro jobs dead-letter list --database-dialect sqlite --database-dsn PATH --json
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
hydro license verify --license hydro.lic --public-key vendor.pub --project-id PROJECT_ID
hydro license status --license hydro.lic --public-key vendor.pub --json
```

`hydro run` is recommended for local development; `hydro dev` remains as a
backwards-compatible alias for older automation.

The default `/` route is a backend runtime status page. It links to health,
readiness, API, and Admin Studio endpoints rather than redirecting to demo
content. For local frontend/browser validation, run `hydro seed-demo
--confirm-demo` and open `/public/pages/{locale}/demo/html`. A configured,
validated `site.home_path` can explicitly attach a product frontend in a real
deployment.

Portable import/export is bounded to 256 MiB by default; use `--max-bytes` when an approved larger customer export is required.

`hydro release verify` does not trust the public key embedded in the manifest by
itself. It requires a trusted Ed25519 public-key file distributed by the operator.

Browser-originated state-changing requests must include a CSRF token matching the
`hydro_csrf` cookie. URL-encoded and small multipart forms may carry the token in
`_csrf`. Large or chunked multipart uploads must send it in `X-CSRF-Token`; this
avoids parsing an unbounded upload body solely for CSRF validation.

Backup and restore are deliberately staging-first. SQLite and PostgreSQL backups
create a new immutable snapshot directory; PostgreSQL uses `pg_dump`/`pg_restore`
with a temporary password file and never places the password in process arguments.
Restore verification checks hashes only, staging never overwrites an existing
destination, and media activation requires the quarantine marker plus all three
operator controls. Pause application writes and background workers before taking
a live snapshot.

Portable import is explicit: `--dry-run` validates the allowlisted, secret-free
document without a database write, while a real import requires `--confirm` and
runs in one transaction.

Snapshot retention is explicit: run `hydro backup prune --dry-run` to inspect
candidates, then repeat with `--confirm`. Pinned snapshot directory names are
never removed.

## Tests and production gates

Daily development checks:

```powershell
go test ./...
go vet ./...
powershell -ExecutionPolicy Bypass -File .\scripts\verify-local.ps1
```

`verify-local.ps1` is the single local developer gate: it checks formatting, vet,
the full test suite, a trimpath build, Client Edition import boundaries, and both
external edition builds. It does not require PostgreSQL. The production-gate
script is stricter and requires a real PostgreSQL contract database.

`hydro doctor --json` provides automation-friendly preflight output for the
selected profile, database, backup tools, media writability, transport security
settings, binary identity, and embedded schema migration level.

Dead-lettered background jobs can be reviewed without exposing their payloads:

```text
hydro jobs dead-letter list --database-dialect sqlite --database-dsn PATH --json
hydro jobs dead-letter requeue --id JOB_ID --confirm --database-dialect sqlite --database-dsn PATH
hydro jobs dead-letter purge --older-than 720h --limit 100 --confirm --database-dialect sqlite --database-dsn PATH
```

Requeue and purge are explicit operator actions. Requeue preserves the job attempt
count and last error; purge is bounded by age and batch size.

After configuring a PostgreSQL contract database, run the automated release gates:

```powershell
$env:HYDRO_TEST_POSTGRES_URL = "postgres://user:password@127.0.0.1:5432/hydro_contract?sslmode=disable"
powershell -ExecutionPolicy Bypass -File .\scripts\verify-production-gates.ps1
```

On Linux or macOS, run the equivalent gate runner:

```bash
export HYDRO_TEST_POSTGRES_URL='postgres://user:password@127.0.0.1:5432/hydro_contract?sslmode=disable'
bash ./scripts/verify-production-gates.sh
```

Both runners check formatting, vet, the full test suite, race-sensitive packages,
reproducible binary builds, and the portable benchmark. They fail when PostgreSQL
is unavailable instead of silently treating contract tests as skipped.

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
- Secure-by-default routing, default-deny authorization, and data portability are Core contracts.

## Documentation

- [Roadmap](docs/ROADMAP.md): phases, open gates, and acceptance criteria
- [Real-world production roadmap](docs/REAL_WORLD_PRODUCTION_ROADMAP.md): the path from pilot to profile-specific production exit gates
- [HydroGo philosophy](docs/HYDROGO_PHILOSOPHY.md): product and architectural principles
- [Implementation plan](docs/implementation_plan.md): scope and target architecture
- [ADR index](docs/architecture/README.md): binding architectural decisions
- [Production gate checklist](docs/validation/PRODUCTION-GATES.md): evidence format for real environments
- [Open gates and operator actions](docs/OPEN_GATES_AND_OPERATOR_ACTIONS.md): operator actions and production blockers
- [Operations runbook](docs/OPERATIONS_RUNBOOK.md): deploy, backup, restore, and alarm flows
- [Support matrix](docs/SUPPORT_MATRIX.md): supported and uncertified profiles
- [Compatibility matrix](docs/COMPATIBILITY_MATRIX.md): old/new release boundaries
- [Threat model](docs/security/THREAT_MODEL.md): security assumptions and open risks
- [Project status](docs/PROJECT_STATUS.md): current capabilities and required evidence
- [Security policy](SECURITY.md): private vulnerability reporting and disclosure guidance
- [Contribution guide](CONTRIBUTING.md) and [repository rules](AGENTS.md)

Dependency vulnerability scanning (`govulncheck`), pinned GitHub Actions, weekly
Dependabot updates, and CodeQL Go analysis run through repository CI. These checks
improve detection and supply-chain integrity; they do not replace the deployment-
specific production drills listed above.

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

Commercial SLA claims for Compact or Standard remain blocked until the R3/R5
evidence gates are closed. Distributed, Redis/Valkey, S3/CDN, OIDC/SAML, advanced
search, commerce, AI, and multi-tenant features have provider-neutral ADR boundaries
but still require real customer/provider evidence and their own verification gates.
Remaining production actions are listed in [`docs/OPEN_GATES_AND_OPERATOR_ACTIONS.md`](docs/OPEN_GATES_AND_OPERATOR_ACTIONS.md).

This repository is proprietary/private and is not published under an open-source
license. The short technical distribution notice is in [`PROPRIETARY_NOTICE.md`](PROPRIETARY_NOTICE.md);
commercial terms remain a separate product decision.
