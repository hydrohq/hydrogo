# 0190. Capacity Profiles, Concurrency Budgets & RPO/RTO Certification

Date: 2026-09-20
Status: Accepted

## Context

HydroGo deployments scale across diverse target hardware and operational tiers—from single-node embedded IoT/edge installations running SQLite to high-traffic distributed clusters operating on multi-node PostgreSQL, Redis/Valkey session stores, and multi-tenant job pools.

To prevent cascading failures, thread exhaustion, unbounded memory consumption, and connection pool starvation:
1. Every runtime environment operates under a deterministic, bounded `Profile` (`compact`, `standard`, `distributed`).
2. Concurrency budgets for database connection pools (`DBMaxOpenConns`, `DBMaxIdleConns`), in-memory cache capacity (`CacheEntries`, `CacheTTL`), worker threads (`WorkerConcurrency`), bounded asynchronous queues (`QueueEntries`), and max request/media body payloads (`HTTPBodyBytes`, `MediaMaxBytes`) must be strictly configured and validated at boot.
3. RPO (Recovery Point Objective) and RTO (Recovery Time Objective) parameters must align with capacity profiles without making fictitious hardware throughput promises.

## Decision

1. **Deterministic Capacity Profiles (`internal/capacity/profile.go`):**
   - Formalize bounded budgets across three profiles:
     - `Compact`: DBMaxOpen=1, DBMaxIdle=1, WorkerConcurrency=1, CacheEntries=256, HTTPMax=64KB, MediaMax=10MB, ShutdownTimeout=10s.
     - `Standard`: DBMaxOpen=10, DBMaxIdle=5, WorkerConcurrency=4, CacheEntries=1024, HTTPMax=1MB, MediaMax=50MB, ShutdownTimeout=20s.
     - `Distributed`: DBMaxOpen=50, DBMaxIdle=25, WorkerConcurrency=16, CacheEntries=4096, HTTPMax=4MB, MediaMax=100MB, ShutdownTimeout=30s.

2. **Fail-Closed Budget Validation:**
   - Boot sequence strictly validates budget parameters (`Budget.Validate()`), rejecting misconfigured connection pools or invalid queue/cache sizes before listener activation.

3. **Graceful Drain & Shutdown:**
   - Enforce deterministic shutdown timeouts (`ShutdownTimeout`) across active HTTP listeners, worker pools, and database connection pools to preserve transaction boundaries and eliminate inflight request corruption.

## Consequences

- **Positive:** Zero risk of uncontrolled memory growth or database connection saturation under spiky loads.
- **Positive:** Predictable resource consumption on constrained compact environments (e.g. Raspberry Pi/VPS).
- **Operational:** Large scale deployments must explicitly select `Standard` or `Distributed` profiles to utilize higher concurrency pool allocations.
