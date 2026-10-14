# R128 — Core dependency/bloat audit — 2026-09-22

## Scope and method

Audited the non-standard Go package closure of the production entry point,
`./cmd/hydro`, on Windows amd64 with Go 1.27.0. Counts are unique imported
packages from `go list -deps`; they are not binary size, RAM, startup latency,
or throughput measurements. The Compact composition was checked with the
existing opt-out set (`hydro_compact,hydro_no_graphql,hydro_no_otlp,
hydro_no_s3,hydro_no_redis`).

Commands:

```powershell
go list -deps -f '{{if not .Standard}}{{.ImportPath}}{{end}}' ./cmd/hydro
go list -deps -f '{{if not .Standard}}{{.ImportPath}}{{end}}' `
  -tags hydro_compact,hydro_no_graphql,hydro_no_otlp,hydro_no_s3,hydro_no_redis ./cmd/hydro
```

## Results

| Composition | Unique non-standard packages |
| --- | ---: |
| Default Workspace | 371 |
| Compact SQLite, without GraphQL/OTLP/S3/Redis | 152 |
| Difference | -219 (-59.0%) |

The resulting Compact binary has separate measured evidence in
[R127](R127-no-redis-runtime-variant-2026-09-22.md): 21,341,696 bytes on this
toolchain/OS/architecture. Neither result predicts runtime memory or customer
performance.

## Dependency ownership findings

`go mod why -m` and source imports identify the prominent remaining groups as
purposeful product capabilities rather than unexplained transitive baggage:

- `golang.org/x/text`: canonical Unicode normalization plus locale-aware
  currency/number formatting (`internal/canonical`, `internal/i18n`).
- `github.com/pquerna/otp` and barcode: MFA enrollment/authentication in
  `internal/auth`.
- `golang.org/x/image/webp`: media MIME/decoder support in `internal/media`.
- `bluemonday`: server-side rich-text sanitization in `internal/ui`.
- OpenTelemetry trace API: provider-neutral request correlation in
  `internal/observability`; the OTLP exporter itself is already removable.

These are security/data-correctness and product contracts, so deleting them
just to lower a package count would be misaligned. The opt-out build matrix
already excludes the large database, GraphQL, OTLP exporter, S3 and Redis
implementations from the Compact artifact.

## Remaining audit question

The package closure still includes media, localization, MFA, sanitization, and
tracing contracts in Compact. The next Core-boundary audit must establish which
are unconditional Compact requirements versus module/profile capabilities, and
prove inactive modules do not retain constructors, routes, migrations, assets,
workers, or mandatory resource overhead. This report alone does not certify
that absence; it records a measured baseline for that work. No production
profile or external Client Edition certification is implied.
