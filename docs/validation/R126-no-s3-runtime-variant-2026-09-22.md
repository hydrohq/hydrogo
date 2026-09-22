# R126 — Optional S3 media provider build — 2026-09-22

## Scope

Measured the Compact SQLite/no-GraphQL/no-OTLP Workspace artifact with and
without S3; verified provider selection and local-media behavior. These local
Windows amd64 results are build/import measurements, not performance or
production-provider certification.

## Results

Go 1.27.0, `-trimpath -buildvcs=false -ldflags "-s -w -buildid="`:

| Artifact | Bytes | MiB | Non-stdlib packages |
| --- | ---: | ---: | ---: |
| Compact SQLite + no GraphQL + no OTLP | 30,108,160 | 28.71 | 196 |
| Same, without S3 (`hydro_no_s3`) | 28,160,000 | 26.86 | 168 |
| Delta from removing S3 | -1,948,160 | -1.86 | -28 |

The S3-free import closure contains no MinIO SDK package. CI enforces ceilings
of 29 MiB and 175 non-standard packages for this Windows Compact variant.
These are regression limits, not customer hardware promises.

## Verification

- `go test ./internal/media ./internal/config ./cmd/hydro -count=1` — passed.
- `go test -tags hydro_no_s3 ./internal/media ./internal/config ./cmd/hydro -count=1` — passed.
- `go test -tags hydro_compact,hydro_no_graphql,hydro_no_otlp,hydro_no_s3 ./internal/media ./internal/config ./cmd/hydro ./internal/database ./internal/tenant ./internal/observability -count=1` — passed.
- `scripts/verify-edition-footprint.ps1` — all six thin Edition variants and the Workspace no-S3 variant passed the declared budgets.
- `scripts/verify-edition-servers.ps1` — both SQLite Edition migration and CRUD smoke tests passed.
- Config preflight rejects S3 configuration in no-S3 builds; local media remains available. Licensed customer package creation was not exercised because signed release credentials were not in scope.

## Remaining gaps

R127 records the later Redis/Valkey separation. Full Client Edition
composition, module-aware command/runtime selection, production startup/RSS/
capacity budgets, and clean deployment certification remain open.
