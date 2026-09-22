# R127 — Optional Redis/Valkey runtime build — 2026-09-22

## Scope

Measured the combined Compact SQLite/no-GraphQL/no-OTLP/no-S3 Workspace
artifact with and without Redis. Verified local cache/limiter fallback and
config rejection of Redis URLs in the no-Redis variant. These are local
Windows amd64 build/import results, not distributed-capacity or production
measurements.

## Results

Go 1.27.0, `-trimpath -buildvcs=false -ldflags "-s -w -buildid="`:

| Artifact | Bytes | MiB | Non-stdlib packages |
| --- | ---: | ---: | ---: |
| Compact SQLite + no GraphQL/OTLP/S3 | 28,161,024 | 26.86 | 168 |
| Same, without Redis (`hydro_no_redis`) | 21,341,696 | 20.35 | 152 |
| Delta from removing Redis | -6,819,328 | -6.50 | -16 |

The no-Redis `cmd/hydro` build closure contains no `github.com/redis/go-redis/v9`
packages. CI enforces a 22 MiB and 160 non-standard-package ceiling for this
Windows Compact artifact. These limits are regression guards, not RAM,
latency, throughput, or customer hardware promises.

## Verification

- Default Redis adapter tests passed in `internal/security`, `internal/cache`,
  and the default `cmd/hydro` composition.
- `go test ./... -p 1 -timeout=5m` — passed for the default build.
- With `HYDRO_REDIS_URL` unset, `go test -tags hydro_no_redis ./... -p 1 -timeout=5m` — passed for the full repository.
- With `HYDRO_REDIS_URL` unset, `go test -tags hydro_no_redis ./cmd/hydro ./internal/config ./internal/cache ./internal/security ./internal/server -count=1` passed.
- The combined Compact/no-Redis tagged tests passed for config, cache, security, server, database, tenant, observability, media and `cmd/hydro`.
- `scripts/verify-edition-footprint.ps1` and Edition SQLite migration/CRUD smoke tests passed.
- Customer runtime scripts accept `HYDRO_REDIS=enabled|disabled` and record the selection. Licensed package creation was not exercised because signed release credentials were not in scope.

## Remaining gaps

This does not certify Redis outage/failover under production load, cross-node
quota behavior, or shared-tenant admission. Full Client Edition composition,
other Core dependency boundaries, and production resource budgets remain open.
