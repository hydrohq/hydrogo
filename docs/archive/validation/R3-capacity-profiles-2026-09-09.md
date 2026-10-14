# R3 capacity profile contract validation

**Date:** 2026-09-09  
**Roadmap:** R3 S05-C / R5 Standard profile preparation  
**Status:** Profile and bounded budget contract succeeded; actual traffic/benchmark certification remains open.

`internal/capacity` now exposes the `compact`, `standard`, and `distributed` profiles from ADR-0007 as a typed contract. Each profile carries a budget for DB pool, L1 cache, worker/queue, HTTP body, media, and graceful shutdown. Defaults are measurement inputs; they are not hardware or throughput guarantees.

`Config` receives the profile via YAML (`profile`), `HYDRO_CAPACITY_PROFILE`, or `--capacity-profile` and rejects unknown profiles at startup. `database.OpenWithOptions` applies pool limits to the profile budget; `hydro serve`, migration, and admin commands use the same profile budget. Compact cache/queue/media values are the same as the previous behavior; Standard and Distributed values are initial measurement assumptions.

`hydro doctor` checks DB ping/checksum compatibility and the media directory with the same profile budget without applying migrations; upon successful check, it prints the selected profile and limits to the operator.

Executed checks:

```text
go test ./internal/capacity ./internal/config ./internal/database ./cmd/hydro
go vet ./internal/capacity ./internal/config ./internal/database ./cmd/hydro
go test ./cmd/hydro -run TestRunMigrateAndCreateAdmin
```

Actual Compact blue/green traffic, p95/p99 and peak memory measurement; Standard PostgreSQL connection pool benchmark; Distributed L2/fan-out and backpressure experiments are outside the scope of this validation.

`BenchmarkCacheCoalesces64ConcurrentLookups` measured 202 µs/op with `-benchtime=10x` on a Windows amd64 development machine, keeping the loader call to 1 for 64 lookups in each iteration. This local benchmark is not a customer capacity/SLA proof; actual traffic and profile soak measurement are also required.
