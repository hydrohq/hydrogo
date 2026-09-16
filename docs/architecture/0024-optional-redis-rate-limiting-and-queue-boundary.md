# ADR-0024: Optional Redis rate limiting and SQL job queue boundary

- **Status:** Accepted for implementation
- **Date:** 2026-09-11
- **Related:** [ADR-0003](0003-cache-architecture.md), [ADR-0007](0007-capacity-profiles.md), [ADR-0011](0011-consistency-upgrade-and-recovery.md)

## Decision

HydroGo supports an optional Redis/Valkey-compatible endpoint through `redis.url` or `HYDRO_REDIS_URL`.
When configured, authentication, recovery, MFA, invitation acceptance, and public form rate limits use an
atomic `INCR` + TTL counter in Redis. Keys contain a fixed SHA-256 digest rather than raw IP, email, or user
identifiers. Redis connection failure is fail-closed for protected attempts; when no endpoint is configured,
the existing bounded process-local limiter remains the Compact default.

The transactional SQL outbox and lease-based job queue remain the source of truth for HydroGo jobs. RabbitMQ is
not added at this stage because it would impose another mandatory operational service and recovery topology on
Compact/Standard deployments. A future broker adapter may be introduced only after a measured multi-worker
throughput requirement and a certified delivery/recovery contract.

## Boundaries

- Redis is coordination/cache infrastructure, never the primary store for content, audit, money, stock, or jobs.
- Redis is selected explicitly; it is not silently started or required by local development.
- The distributed limiter uses bounded key prefixes and TTLs; it does not trust client-provided forwarded IP headers.
- Scale-out deployments must validate Redis/Valkey availability, persistence policy, outage behavior, and keyspace isolation.

## Verification

- Local limiter and Redis limiter share the `RateLimiter` application boundary.
- Atomic counter semantics are implemented by a single Redis Lua script.
- Redis is connectivity-checked at startup when configured.
- Request-scoped callers use `AllowContext`; cancellation and deadlines are
  propagated while retaining the 250 ms fail-closed safety bound.
- The SQL queue remains covered by the existing SQLite/PostgreSQL contract tests.
