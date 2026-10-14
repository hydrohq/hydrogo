# HydroGo Core Bloat Audit

**Date:** 2026-09-22  
**Status:** Baseline recorded; remediation is tracked by the platform-factory roadmap P0.

## Scope

This audit checks whether the single `hydro` binary and the `internal/` Core
packages carry optional product capabilities into the Compact profile. It does
not treat a single binary as a defect by itself: the question is whether an
unused capability creates avoidable binary, startup, memory, dependency, route,
migration, worker, or operational cost.

## Baseline evidence

Commands:

```text
go build -o .tmp/hydro-audit.exe ./cmd/hydro
go build -ldflags="-s -w" -o .tmp/hydro-audit-stripped.exe ./cmd/hydro
```

Observed on the Windows development host:

| Artifact | Size |
|---|---:|
| Unstripped `hydro` | 54,924,288 bytes |
| Stripped `hydro` | 38,728,192 bytes |

The `cmd/hydro` dependency graph currently includes these optional-capability
families:

- SQLite (`modernc.org/sqlite`) and PostgreSQL (`pgx/v5`)
- Redis (`go-redis/v9`)
- S3/MinIO (`minio-go/v7`)
- GraphQL (`graphql-go/graphql`)
- OpenTelemetry SDK and OTLP HTTP exporter
- Templ and server-rendered admin/UI assets
- image/WebP processing (`golang.org/x/image`)

This confirms that a default build carries more than a minimal content-only
server needs. It is not yet evidence that all of these packages are active at
runtime; route, worker and connection activation must be checked separately.

## Findings

### F-01 — Composition root imports optional capabilities

`cmd/hydro/main.go` imports database, cache, media, telemetry, GraphQL, tenant,
admin UI and operational packages in one executable composition root. This
keeps distribution simple, but it makes binary-level exclusion impossible
without profile-aware composition or build-time capability selection.

**Risk:** Compact deployments pay binary and attack-surface cost for disabled
features.  
**Severity:** Medium (operational/resource), potentially High for strict
customer distribution requirements.

### F-02 — Profile activation is not yet proven as a zero-footprint property

Existing tests verify disabled modules do not register routes/resources in
several paths, but there is no single gate proving the complete capability
matrix: imports, assets, migrations, workers, goroutines, pools and startup
latency for every profile.

**Risk:** A feature can be “disabled” functionally while still consuming
resources or remaining reachable in the binary.  
**Severity:** Medium.

### F-03 — One binary does not imply one runtime dependency set

The single-binary contract is valuable for deployment, but it must be paired
with a capability manifest. Otherwise “single binary” can hide a growing
monolith and make supply-chain review harder.

**Risk:** Upgrade, SBOM, CVE triage and customer-specific distribution become
less predictable.  
**Severity:** Medium.

## Remediation plan

1. Introduce a versioned capability registry with explicit profile manifests.
2. Move optional composition behind module/kit boundaries; keep interfaces in
   Core and providers in opt-in packages.
3. Add `hydro profile explain` and `hydro profile check` to report active
   routes, migrations, workers, assets, pools, goroutines and dependencies.
4. Add Compact/Standard/Distributed startup and resource budgets.
5. Add a release gate that compares binary/SBOM/resource deltas and rejects
   unexplained growth.
6. Preserve the single-binary distribution option; do not reintroduce a
   runtime plugin loader or customer-specific Core forks.

## Acceptance criteria

- Compact profile has no Redis, S3, GraphQL or distributed-worker activation
  unless explicitly selected.
- Disabled capability creates no route, migration, asset, worker, pool or
  mandatory goroutine.
- Every profile has a versioned capability manifest and resource budget.
- Binary/SBOM growth is reviewed by a commit/ADR, not accepted implicitly.
- SQLite/PostgreSQL behavior and blue/green compatibility remain certified.

## Non-goals

- Splitting HydroGo into microservices.
- Removing PostgreSQL, Redis, S3, GraphQL or observability support.
- Sacrificing the single-binary deployment contract.
