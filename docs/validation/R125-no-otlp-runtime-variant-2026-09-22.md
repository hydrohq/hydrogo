# R125 — Optional OTLP exporter runtime build — 2026-09-22

## Scope

Measured the combined Compact SQLite/no-GraphQL/no-OTLP Workspace artifact,
tested default and exporter-free behavior, and inspected the Go dependency
closure. This is a local binary/import measurement, not an app throughput,
latency, RSS, or production workload benchmark.

## Results

Windows amd64, Go 1.27.0, stripped build:

| Artifact | Bytes | MiB | Non-stdlib packages |
| --- | ---: | ---: | ---: |
| Compact SQLite + no GraphQL, OTLP present | 34,383,360 | 32.79 | 337 |
| Compact SQLite + no GraphQL + no OTLP | 30,108,160 | 28.71 | 196 |
| Delta from removing OTLP | -4,275,200 | -4.08 | -141 |

Compared with the default binary baseline from R120 (36.98 MiB / 371
non-standard packages), the combined variant is materially smaller. The
difference is not a pure one-feature benchmark because it excludes both
PostgreSQL and GraphQL as well as OTLP.

The final variant's `go list -deps` closure contains no
`go.opentelemetry.io/otel/exporters/otlp`, OTLP protobuf, or gRPC package.
The Go module graph still declares dependencies needed by other build/test
variants; this change does not make repository-wide supply-chain tooling forget
those modules.

CI enforces a ceiling of 32 MiB and 210 non-standard packages for the Windows
Compact Workspace variant. This regression ceiling is not a production-size
promise.

## Verification

- `go test ./internal/observability ./cmd/hydro -count=1` — passed.
- `go test -tags hydro_no_otlp ./internal/observability ./cmd/hydro -count=1` — passed.
- `go test -tags hydro_compact,hydro_no_graphql,hydro_no_otlp ./internal/observability ./cmd/hydro ./internal/config ./internal/database ./internal/tenant -count=1` — passed.
- Import closure inspection found no OTLP exporter, OTLP protobuf, or gRPC packages in the final variant.
- Customer PowerShell/Bash packagers support `HYDRO_OTLP=enabled|disabled` and record the selected option. Licensed package generation was not exercised because signed customer release credentials were not in scope.

## Remaining gaps

Redis/Valkey and other optional runtime features remain candidates for
composition-time selection. R126 records the later S3 provider separation.
Full Client Edition composition, production startup/RSS/capacity measurements,
and deployment certification remain open.
