# 0191. Distributed Cache & Multi-Instance Session Failover Certification

Date: 2026-09-20
Status: Accepted

## Context

When HydroGo nodes scale horizontally in multi-instance or high-availability deployments:
1. Local in-memory caches (L1) must invalidate deterministically without serving stale domain entity projections across nodes.
2. Two-tier caching (L1 local in-memory + L2 distributed remote) must coalesce concurrent loads (stampede / thundering herd protection), gracefully fall back to database loaders if L2 is temporarily unreachable, and enforce tenant namespace isolation on every key.
3. Multi-instance BFF sessions and web authentication state must persist across horizontal node restarts and load-balancer failovers via encrypted Redis/Valkey session backends (`internal/security/bffsession`).
4. SQLite deployments running in multi-process configurations must detect external WAL/data-version mutations via `SQLiteDataVersionMonitor` and invalidate L1 caches immediately.

## Decision

1. **Stampede-Protected Two-Tier Cache (`internal/cache`):**
   - Implement generational coalesced loaders (`GetOrLoad`, `GetOrLoadAtGeneration`) preventing duplicate backend queries.
   - Fallback to safe local database load with L1 warming when L2 encounters network or serialization failures.
   - Enforce explicit tenant scoping on cache keys (`Scoped`), preventing multi-tenant data leakage.

2. **Distributed & Encrypted BFF Session Store (`internal/security/bffsession`):**
   - Provide AES-GCM encrypted Redis/Valkey session backend with strict cryptographic key validation (rejecting short/weak keys).
   - Server-side cookie abstraction isolating upstream HydroCore authentication cookies from untrusted browser surfaces and validating double-submit CSRF tokens.

3. **External Write Cache Monitor (`internal/cache/sqlite_monitor.go`):**
   - Poll `PRAGMA data_version` with exponential backoff on errors, clearing local caches whenever external writes occur.

## Consequences

- **Positive:** Linear scalability across multiple backend replicas with zero cache inconsistency or stampede risk.
- **Positive:** Resilient failover during pod restarts; users remain seamlessly logged in via distributed encrypted session store.
- **Operational:** Shared Redis/Valkey deployment is optional in single-node compact setups and mandatory for distributed multi-instance topologies.
