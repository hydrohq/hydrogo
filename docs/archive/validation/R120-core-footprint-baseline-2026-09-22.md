# R120 — Core footprint and import-graph baseline

Date: 2026-09-22

## Measurement

On the Windows amd64 development host, `go build -trimpath -buildvcs=false
-ldflags "-s -w -buildid=" -o <temp> ./cmd/hydro` produced a 38,776,320-byte
(36.98 MiB) executable. `go list` reports 54 packages under `internal/`, 148
module-graph entries, 371 non-standard-library packages reachable from
`cmd/hydro`, and 623 total dependency packages in that command's build graph.
These are a local baseline, not a cross-platform release-size guarantee.

The production entry's direct imports include the server/runtime, API and
admin modules, project/resource generator, SDK distribution/release commands,
both SQL drivers (SQLite and PostgreSQL through `internal/database`), Redis,
and GraphQL. Reachability traces confirm that MinIO is imported by
`internal/media`, OTLP/GRPC exporters by `internal/observability`, and Redis
directly by `cmd/hydro`. They are linked even when a given deployment does not
configure those capabilities. Go's linker removes dead code but does not omit
entire directly imported packages merely because a runtime profile leaves them
disabled.

## Package boundary inventory

- **Runtime foundations:** `config`, `database`, `modules`, `router`,
  `requestmeta`, `pathguard`, `guards`, `server`, `security`, `tenant`.
- **Composable runtime capabilities:** `auth`, `cache`, `content`, `media`,
  `jobs`, `outbox`, `mailer`, `observability`, `telemetry`, `backup`,
  `webhooks`, `forms`, `analytics`, `search`, `taxonomy`, `canonical`,
  `eventcatalog`, `erasure`, `audit`, `capacity`.
- **Persistence adapters:** `auth/sqlstore`, `audit/sqlstore`,
  `analytics/sqlstore`, `forms/sqlstore`, `jobs/sqlstore`, `media/sqlstore`,
  `outbox/sqlstore`.
- **Admin/UI:** `adminstudio`, `ui`, `ui/components`, `i18n`.
- **Workspace/release tooling:** `generator`, `projectgen`, `sdkdist`,
  `release`, `portable`, `licensing`, `ops`, `extensionpkg`.
- **Versioned SDK/support contracts:** `extension`, `contracts/customerorders`,
  plus nested `security/bffpolicy`, `security/bffproxy`, and `security/bffsession`.

Classification is based on package responsibility; it does not imply that a
capability has a stable public API or belongs in every Edition.

## Existing evidence and gaps

Existing tests verify that inactive modules do not contribute routes or
registry manifests and that an inactive Pages module does not construct its
page cache. Existing `internal/capacity` profiles bound DB pools, cache,
workers, queues and request/media bodies. `hydro profile explain` and
`hydro profile check` already exist. They expose bounded defaults and a
hand-authored capability manifest, but that manifest explicitly leaves actual
provider/route binding to composition roots; it does not mechanically resolve
an Edition or prove its linked/runtime capability set. There is no per-Edition
binary/import budget. Existing tests cover selected route and Pages-cache
non-construction, not zero inactive migration, asset, worker, goroutine, or
all adapter construction across every module.

Thin generated-route Edition harnesses provide a narrower composition path. A
stripped Windows amd64 build of both
`examples/project-edition/cmd/server` and
`examples/article-edition/cmd/server` measured 15,782,400 bytes with 89
non-standard-library dependency packages each. This is materially smaller than
the universal binary, but these harnesses expose only a generated route and a
health check; they are not the full Admin/Identity/Site production surface.
R122 turns this limited result into a Windows CI regression gate.

No dependency was removed in this baseline-only stage. The broad R110
over-engineering audit found no safe unconditional deletion; this measurement
identifies a different issue: optional capabilities are still present in the
single universal executable. Do not blindly remove a supported capability to
chase a size number.

## Recommended P0 follow-up

1. Extend the thin Edition composition path to a complete Client Edition from
   its selected module manifest. Keep workspace generation/release tooling
   out of the deployed Edition binary and retain only selected operator
   commands, migrations and providers.
2. Make driver/exporter choices composition-time, so SQLite Compact need not
   link PostgreSQL, Redis, S3, OTLP/GRPC, or GraphQL unless that Edition opts in.
   Retain shared interfaces and certify each supported combination.
3. Bind `hydro profile explain/check` to the Edition's resolved capability
   graph and add measurable budgets for binary bytes, startup, steady-state
   RSS, connections, workers, and inactive-module construction. Re-measure
   Linux/Windows full Edition artifacts before accepting a product size target.

This report is a baseline, not P0 completion. ADR-0247 accepts only the narrow
fixture build budget; ADR-0248/R123 additionally measure a SQLite-only build,
but full Edition composition and profile budgets still need an ADR-backed
compatibility proof.
