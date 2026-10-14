# ADR-0003: Multi-tier professional cache

- **Status:** Superseded (partially)
- **Date:** 2026-09-05
- **Superseded by provisions in:** [ADR-0011](0011-consistency-upgrade-and-recovery.md)

> L0–L3 tiers and profile selection remain valid. The following general uninterruptedness and worker-L1 invalidation statements have become historical; for multi-process fan-out, generation, stale budget, and bounded fallback, ADR-0011 applies.

## Context

HydroGo must run efficiently on a small single server without additional services and on multi-instance setups. HTTP cache is not a legacy technology; defined by the current standard RFC 9111, it is the cheapest cache tier that fulfills the request at the browser and edge layer before reaching the application. Distributed cache solves a different problem, sharing and coordination between instances.

## Decision

The cache is designed in four tiers:

```text
L0  Request-scope memoization
L1  HydroGo bounded memory cache
L2  Redis/Valkey compatible distributed cache (optional)
L3  HTTP reverse proxy / CDN / browser cache
```

| Profile | L1 Memory | L2 Distributed | L3 HTTP/CDN |
|---|---:|---:|---:|
| Compact | Yes | No | Yes |
| Standard | Yes | Optional | Yes |
| Distributed | Yes | Mandatory | Mandatory |

Valkey, being BSD licensed and backed by the Linux Foundation, is the default self-hosted distributed cache candidate. An adapter compatible with the Redis protocol is provided. Since Redis 8's license options carry different obligations, versions and licenses are evaluated separately in product distribution.

## Cache contract

The cache contains only reproducible data. Cache loss does not turn into data loss or application downtime.

```go
type Cache interface {
    Get(ctx context.Context, key string) ([]byte, bool, error)
    Set(ctx context.Context, key string, value []byte, ttl time.Duration, tags ...string) error
    Delete(ctx context.Context, keys ...string) error
    InvalidateTags(ctx context.Context, tags ...string) error
}
```

Planned drivers:

- `NoopCache`
- `MemoryCache`
- `ValkeyCache`
- `RedisCache`

Although Redis and Valkey share the same protocol, their names are explicitly stated in product configuration due to behavior, version, and license differences.

## Cacheable data

- Published public HTML
- Public API responses
- Menu and navigation tree
- Public subset of site settings
- SEO, sitemap, and RSS outputs
- Expensive to calculate, reproducible query results

## Non-cacheable data

- Password, token, and sensitive session content
- Audit events
- Single source of truth for jobs
- Drafts and private previews
- Primary record of money, stock, and payment ledger
- Long-lived result of an authorization decision

## Professional cache behavior

- Cache-aside is the primary read pattern.
- Each key is namespaced with product, customer, locale, content type, version, and serialization version.
- TTL is mandatory and jitter is added to prevent simultaneous mass expiration.
- Cache stampede is suppressed with `singleflight` in a single instance; in a distributed profile, it is bounded with short lease/lock.
- Negative cache is used only in safe `not found` results and with a short TTL.
- Large inputs are rejected with `max_entry_size`.
- L1 memory is byte-budgeted and bounded; unbounded maps are not used.
- If the distributed cache disconnects, the application safely falls back to DB in a controlled manner and applies circuit breaker/backoff.
- Cache failure does not roll back the content write transaction.
- Hit/miss, latency, eviction, entry bytes, stampede suppression, and invalidation failure are measured.

## Invalidation

In addition to key deletion, versioned namespaces are used:

```text
page:{site}:{locale}:{slug}:{published_revision}
menu:{site}:{locale}:{menu_version}
collection:{site}:{locale}:{type}:{publication_version}:{query_hash}
```

The publication transaction writes an outbox event along with the new revision/version. After the commit, a worker:

1. Clears the L1 cache.
2. Sends L2 tag/namespace invalidation.
3. Performs CDN purge or surrogate-key invalidation.

Failed invalidation is retried. TTL and versioned keys limit the persistent stale data risk.

## HTTP cache

- Only public `GET/HEAD` responses are suitable for shared cache.
- Admin, preview, and authenticated responses will be `private, no-store`.
- ETag and conditional requests are supported.
- If HTMX fragment and full page return from the same URL, `Vary: HX-Request` is used.
- Hashed assets are cached as `immutable`.
- Assets referenced by old HTML are kept longer than the cache lifetime.

HTTP cache is not marketed alone as "the cache system" in the product; it is part of the same observable cache architecture with edge, memory, and distributed cache.

## References

- [RFC 9111 — HTTP Caching](https://www.rfc-editor.org/rfc/rfc9111.html)
- [Valkey overview](https://valkey.io/)
- [Redis licenses](https://redis.io/legal/licenses/)
