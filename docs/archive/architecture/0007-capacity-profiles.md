# ADR-0007: Capacity profiles without numerical hardware promise

- **Status:** Accepted
- **Date:** 2026-09-05
- **Supersedes:** [ADR-0004](0004-scaling-profiles.md)

## Context

Making specific RAM numbers a product promise is misleading. With the same amount of RAM, completely different results are produced by CPU, disk, network, data size, and traffic shape. HydroGo should remain simple in small setups and be able to switch to a different topology with the same job/module contracts when needs grow.

## Decision

Instead of numerical hardware packages, HydroGo provides three capacity/topology profiles:

| Profile | Topology | DB | Cache | Media |
|---|---|---|---|---|
| Compact | Single process or same binary web+worker | SQLite | Bounded memory + HTTP | Local immutable disk |
| Standard | Single or separated web/worker | PostgreSQL; SQLite if appropriate | Memory, distributed cache optional | Local or object storage |
| Distributed | Multiple web and worker | PostgreSQL HA/managed | Memory + Valkey/Redis + HTTP/CDN | Object storage + CDN |

The profile defines the system topology, not the hardware class.

## Resource management

Each profile and edition provides a default based on the actual benchmark result:

- Go memory limit
- L1 cache byte budget
- DB connection pool
- Worker concurrency
- Media decode/encode concurrency
- HTTP in-flight and body limits
- Queue/backpressure thresholds
- Graceful shutdown duration

The user can explicitly override them. `hydro doctor` checks the total budget according to the visible physical/cgroup limit, CPU, and disk conditions and gives a recommendation; the application does not secretly change the topology or database.

## Scaling path

```text
Compact
  → switch to PostgreSQL
Standard
  → object storage + distributed cache
Distributed
  → web/worker horizontal scaling
```

Public traffic is first offloaded from the origin with proper HTTP cache and CDN. The database or the number of instances is increased only if measurements require it.

## Product claim

HydroGo does not promise a fixed amount of RAM or number of concurrent users. For each release/edition, it publishes the following values with a reference workload:

- Hardware and topology
- Dataset
- Cache hot/cold state
- Read/write ratio
- p50/p95/p99 latency
- Throughput and error rate
- Peak/idle memory

## Consequences

- The marketing claim remains measurable.
- Small setups do not bear distributed infrastructure overhead.
- Large setups grow with the same domain and module contracts.
- The support boundary is understood not by a hardware label, but by the published benchmark and certified topology.
