# R124 — Optional GraphQL runtime build — 2026-09-22

## Scope

Measured a Workspace Compact+SQLite artifact with GraphQL omitted and verified
that default GraphQL behavior remains. This is a dependency/size check, not a
throughput benchmark, production load test, or complete Edition measurement.

## Results

Windows amd64, Go 1.27.0, `-trimpath -buildvcs=false -ldflags "-s -w -buildid="`:

| Workspace artifact | Bytes | MiB | Non-stdlib packages |
| --- | ---: | ---: | ---: |
| SQLite-only (`hydro_compact`) | 35,099,648 | 33.47 | 349 |
| SQLite-only without GraphQL (`hydro_compact,hydro_no_graphql`) | 34,383,360 | 32.79 | 337 |
| Delta | -716,288 | -0.68 | -12 |

`go list -deps` confirms there is no HydroGo GraphQL handler package or
`github.com/graphql-go/graphql` parser package in the no-GraphQL build. The
default build still compiles and the route stays available when Pages is
enabled. The route is omitted from the no-GraphQL build.

The generated Project/Article fixture servers measured the same 11,661,312 B /
60 packages in SQLite-only and SQLite/no-GraphQL builds because those narrow
fixtures never imported GraphQL. They remain harnesses, not full customer
Editions.

R125 subsequently removes the independent OTLP exporter tree and records the
combined smaller artifact; this report retains the isolated GraphQL delta.

## Verification

- `go test ./cmd/hydro -count=1` — passed; default feature retained.
- `go test -tags hydro_no_graphql ./cmd/hydro -count=1` — passed; handler absent.
- `scripts/verify-edition-footprint.ps1` — all six Edition variants and the
  Compact SQLite/no-GraphQL Workspace build passed (34 MiB / 340-package caps).
- `scripts/verify-edition-servers.ps1` — both SQLite migration and CRUD smoke
  checks passed.
- Customer runtime scripts accept `HYDRO_GRAPHQL=enabled|disabled` and record
  the chosen setting. Licensed package generation was not exercised because
  signed customer release credentials were not in scope.

## Remaining gaps

This is one independently selectable feature. Redis/Valkey, S3/MinIO, OTLP,
and other optional capabilities require their own composition decisions,
compatibility checks, and measured variants. Full Edition composition and
production deployment remain unverified.
