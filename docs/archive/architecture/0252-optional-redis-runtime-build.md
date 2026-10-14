# ADR-0252: Optional Redis/Valkey runtime build capability

- **Status:** Accepted
- **Date:** 2026-09-22
- **Related:** ADR-0003, ADR-0007, ADR-0010, ADR-0024, ADR-0247, ADR-0248

## Context

ADR-0024 makes Redis/Valkey optional: Compact uses bounded process-local rate
limits and cache by default; configured distributed limits/cache require an
available Redis endpoint. Despite this, the universal Workspace executable
always linked go-redis through the runtime composition root, cache remote, and
distributed rate/admission adapters. Editions that do not use shared
coordination should not carry a Redis client, pool, or network dependency.

## Decision

The default build retains Redis/Valkey support. An Edition can exclude it
using `hydro_no_redis`; customer package builders accept
`HYDRO_REDIS=enabled|disabled` (default `enabled`) and record the capability
in `PACKAGE-MANIFEST.md`.

In a no-Redis build, configuration with a non-empty `redis.url` fails during
config validation. Runtime composition uses the bounded local rate limiter
and local page cache. It does not silently ignore a requested distributed
store. The distributed tenant admission stub always denies tenant-scoped
admission if directly injected, preserving fail-closed behavior. Queue and
job durability continue to use SQL and are unaffected.

The no-Redis artifact must omit `github.com/redis/go-redis/v9` and its
transitive packages from the runtime dependency closure. This is static
Edition composition, not a runtime plugin system.

## Alternatives

- Keep the client linked but instantiate it only when a URL is configured:
  preserves runtime flexibility but leaves binary and supply-chain overhead.
- Silently use local rate limits/cache when `redis.url` is configured:
  rejected because multi-instance quota/security semantics would change
  without operator visibility.
- Remove Redis support globally: rejected; Standard/Distributed Editions may
  require shared rate limits, cache, or tenant admission.

## Consequences

### Benefits

- Editions without Redis carry no go-redis runtime client or pool.
- Compact local cache and rate-limit behavior remains bounded and available.
- A configured distributed capability fails before serving when not linked.

### Costs and risks

- Redis/Valkey support is fixed at build time; artifact and config must match.
- Local rate limits are process-local and are not equivalent to distributed
  limits across replicas. Shared tenancy/capacity must not be enabled under a
  no-Redis build where distributed admission is required.
- Redis remains a module-graph dependency for other builds and tests; this
  decision removes it from selected runtime artifacts, not from `go.mod`.

## Scale impact

- Compact: local cache/limiters only; no Redis connection pool or network work.
- Standard: opt into Redis by selecting an enabled artifact and URL.
- Distributed: Redis is required where quotas or shared cache need cross-node
  consistency; select the provider-enabled build and certify the service.

## Database impact

None. SQL outbox/job lease source of truth, migration ledger, SQLite/PostgreSQL
contracts, and transaction boundaries do not change.

## Cache impact

No-Redis build uses the existing bounded local page cache. Redis-enabled builds
retain the two-tier local/remote cache. Cache keys, TTL, generation
invalidation, stampede protection, and fallback behavior are unchanged.

## Deployment and compatibility

No schema migration. A non-empty `HYDRO_REDIS_URL` with a no-Redis artifact
fails config validation. Blue/green instances must use builds matching the
shared-cache/rate-limit topology; a no-Redis binary must not replace a provider
required for cross-node quotas. Rollback requires the previous artifact and
compatible environment settings.

## Validation

CI tests default Redis adapters, no-Redis fail-closed config, local limiter
fallback, and builds the Compact Workspace variant. `go list -deps` must show
no go-redis packages; artifact and import budgets are recorded in R127.

## References

- [Optional Redis rate limiting and SQL job queue boundary](0024-optional-redis-rate-limiting-and-queue-boundary.md)
- [Compact runtime dependency measurements](../validation/R127-no-redis-runtime-variant-2026-09-22.md)
