# ADR-0247: Generated Edition footprint regression budget

- **Status:** Accepted
- **Date:** 2026-09-22
- **Related:** ADR-0007, ADR-0010, ADR-0143, ADR-0144, ADR-0214

## Context

ADR-0010 requires selected Edition builds to import only chosen modules, and
ADR-0214 demonstrates thin standalone generated-resource servers. They differ
from the universal Workspace/runtime binary: a stripped Windows amd64
`cmd/hydro` build is 36.98 MiB, while generated Project/Article route servers
are currently about 15.05 MiB. The narrow Edition fixture needs a regression
gate without being misrepresented as a complete production Site Edition.

## Decision

On Windows amd64 with the repository-pinned Go 1.27.0 toolchain, each
`project-edition` and `article-edition` thin server verification target must:

- build as one executable with `-trimpath -buildvcs=false -ldflags "-s -w -buildid="`;
- remain at or below 18 MiB;
- have no more than 100 non-standard-library packages in its Go build closure;
- exclude Workspace CLI, generator/project/release tooling, GraphQL, MinIO,
  Redis, and OTLP/gRPC exporter packages.

CI enforces this budget. It is an internal artifact regression target, not a
customer hardware, latency, RAM, or production-runtime promise. It applies
only to the thin generated-route verification Editions. The universal `hydro`
executable remains available for Workspace and operator workflows.

## Consequences

- Thin Edition builds remain single self-contained binaries with composition
  time module selection; runtime plugins and runtime DDL are not introduced.
- The default fixtures retain both SQLite and PostgreSQL drivers for contract
  coverage; ADR-0248 adds an explicitly measured SQLite-only variant.
- Binary size and import closure are both checked because dead-code removal
  can otherwise hide an unintended dependency.
- The budget does not certify Admin, Identity, Site, multi-tenant, or
  production Edition composition. Expanding it requires new evidence and an
  explicit ADR update.
- This build/CI contract creates no additional Compact service, connection
  pool, or goroutine. Cache invalidation and schema behavior are unchanged.
- Both migrations remain embedded and operator-invoked; `serve` performs no
  DDL. Rollback remains an application rollback/forward migration, not a
  destructive schema reversal.

## Verification

`scripts/verify-edition-footprint.ps1` builds both examples in dual and
SQLite-only variants and fails on byte, package-count, or forbidden-import
regressions. Historical dual-build results are in R122; current variant
measurements are in R123. Full Client Edition composition and production
profile budgets remain open.
