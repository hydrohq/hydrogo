# ADR-0004: 512 MB–128 GB resource profiles

- **Status:** Superseded — [ADR-0007](0007-capacity-profiles.md)
- **Date:** 2026-09-05

## Context

HydroGo must be able to run on a low-cost VPS and meaningfully utilize additional resources on larger hardware. This goal is achieved not with a single auto-configuration, but with explicit resource profiles and measured boundaries.

## Decision

HydroGo provides four resource profiles. The values are initial budgets; actual defaults are verified by benchmark.

| Profile | RAM | Application | DB | Cache | Media |
|---|---:|---|---|---|---|
| Nano | 512 MB–1 GB | Single process/instance | SQLite | 16–64 MB L1 | Local immutable disk |
| Standard | 2–8 GB | Single instance, can have separate web/worker | SQLite, MySQL or PostgreSQL | 64–512 MB L1, L2 optional | Local or object storage |
| Performance | 16–32 GB | Blue/green, multiple web instances | PostgreSQL/MySQL | L1 + Valkey/Redis | Object storage + CDN |
| Scale-out | 64–128 GB+ total capacity | Multiple instances and workers | PostgreSQL/MySQL HA | Distributed cache mandatory | Object storage + CDN |

Supporting 128 GB does not mean a single process consumes all memory. The load is distributed among web instances, workers, distributed cache, database, and media processing.

## Resource budgets

Each edition publishes the following budgets:

- Idle RSS
- p50/p95/p99 latency under defined traffic
- Maximum L1 cache bytes
- DB connection limit
- Worker concurrency
- Request body and upload limit
- Concurrent image transformation
- Graceful shutdown duration

In the Nano profile, the target idle RSS value of Core + Site Edition is kept under 128 MB; the remaining memory is for the OS, SQLite page cache, requests, and a safe peak margin. This is a release gate and is measured on the actual binary.

## Configuration

The profile is explicitly selected:

```yaml
runtime:
  profile: nano
```

`hydro doctor` reads the machine's visible physical/cgroup limit and warns if the selected profile is inappropriate. The application does not secretly change the database type or topology.

The profile determines the following defaults:

- Safe upper bound for `GOMEMLIMIT`
- L1 cache budget
- DB pool size
- Worker and media concurrency
- Maximum upload and response buffer
- Timeout and queue backpressure thresholds

The user can override these values; `hydro doctor` reports a dangerous total budget.

## Vertical and horizontal scaling

- The SQLite profile is first scaled vertically, and public traffic is offloaded from the origin with a CDN.
- When a second active web instance is needed for the same customer, a switch to the PostgreSQL/MySQL + distributed cache profile is made.
- Multiple instances are not launched unless media is migrated from local disk to object storage.
- The number of workers is not increased before idempotent jobs and the lease contract are verified.

## Backpressure

Large machine support does not mean unlimited concurrency. Every expensive resource is bounded:

- HTTP in-flight requests
- DB connections
- Image decode/encode jobs
- Email/webhook workers
- Export/import jobs
- Cache entries and total bytes

When the limit is exceeded, instead of continuing to consume memory, the system queues, returns a controlled `429/503`, or places the job in a persistent queue.

## Validation

Each release is tested at least at two extremes:

1. Nano soak test under a 512 MB memory limit.
2. Concurrency and throughput test in a high core/RAM environment.

Performance claims are not published without specifying the request type, data size, cache state, and hardware.
