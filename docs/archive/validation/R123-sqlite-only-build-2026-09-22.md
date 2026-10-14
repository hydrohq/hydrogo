# R123 — SQLite-only build variant — 2026-09-22

## Scope

Validated the `hydro_compact` build tag, configuration fail-closed behavior,
thin Edition footprint budgets, and SQLite smoke behavior. This is a
build/dependency measurement, not an application performance benchmark or
production deployment certification.

## Results

Windows amd64, repository-pinned Go 1.27.0, stripped artifacts:

| Artifact | Dual DB | SQLite-only | Change |
| --- | ---: | ---: | ---: |
| Thin Project/Article Edition | 15,783,936 B / 89 non-stdlib packages | 11,661,312 B / 60 packages | -4,122,624 B / -29 packages |
| Universal `cmd/hydro` | 38,777,856 B / 371 non-stdlib packages | 35,094,016 B / 349 packages | -3,683,840 B / -22 packages |

The SQLite-only universal binary has no pgx build-closure packages. Its
remaining size demonstrates that database driver removal alone does not solve
universal Core/Workspace composition bloat; MinIO, Redis/Valkey, telemetry, and
other shared runtime dependencies remain separate P0 candidates. These sizes
are not RAM, latency, throughput, or production hardware promises.

## Checks

- `go test ./internal/database ./internal/tenant -count=1` — passed.
- `go test -tags hydro_compact ./internal/database ./internal/tenant ./internal/config -count=1` — passed; config preflight rejects PostgreSQL when pgx is not linked.
- `scripts/verify-edition-footprint.ps1` — dual and SQLite-only variants passed for both generated fixtures.
- `scripts/verify-edition-servers.ps1` — SQLite migration, health, and generated CRUD smoke checks passed.
- Customer package builders support `HYDRO_DATABASE_VARIANT=dual|sqlite`, default `dual`; licensed customer package creation was not exercised because signed release credentials were not in scope.

## Remaining evidence gaps

- This does not validate production load, tenant isolation under live PostgreSQL,
  or complete customer Edition composition.
- Universal binary still links substantial optional runtime dependencies.
- PostgreSQL-capable Editions must continue to use the dual-driver artifact.
