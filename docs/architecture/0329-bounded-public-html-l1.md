# ADR-0329: Bounded public HTML process cache

- **Status:** Accepted
- **Date:** 2026-09-29
- **Supersedes:** None
- **Superseded by:** None

## Context

Public rendered pages can repeat expensive rendering work. Existing page data
caching and CGI disk snapshots do not serve a process-local rendered response
for normal HTTP operation. Shared response caching must not retain drafts,
private previews, authenticated output, fragments, or unbounded response data.

## Decision

Add an optional process-local L1 for allowlisted public page/news HTML routes.
The cache stores complete `200 text/html` responses only, with a maximum of 128
entries, 32 MiB total body data, 512 KiB per body, and a default 30-second TTL.
Cookies other than the existing explicitly public consent cookies, authorization,
query strings, HTMX requests, private/no-store/no-cache responses, and
unsupported `Vary` responses bypass storage. ETags and conditional requests are
served from the cached representation. HTML and CSP nonce placeholders are
regenerated together on every response;
compressed bytes are produced only after the cache middleware so a nonce is
never replayed from a stored gzip/Brotli body.

Page publish purges its route and site configuration mutations purge both
content tags. News mutation routes purge the news tag. Generation checks prevent
a local render started before invalidation from repopulating the cache.

This is **process-local only**. It does not claim cross-process fan-out, durable
invalidation, edge/browser recall, or zero stale content. Those guarantees
require the generation/outbox reconciliation protocol from ADR-0011 and remain
outside this decision. Do not enable this layer as the correctness mechanism
for routes that require rapid revocation.

## Database and migration impact

SQLite, PostgreSQL, and future database adapters are unchanged. No schema,
migration, worker, or external service is introduced.

## Cache and capacity impact

The cache is bounded independently of request volume. Expiration and LRU-like
oldest-expiry eviction cap retained response memory. Concurrent misses may
render more than once; normal request admission and database budgets still
apply. A process restart clears the cache. Every process owns its own configured
budget, so deployment capacity planning multiplies the 32 MiB maximum by the
number of web processes.

## Deployment and compatibility

The cache is built into the existing binary and adds no runtime dependency.
Deploys and process restarts safely lose entries. Existing routes and stored
content are unchanged. Response cache headers are short lived; public clients
may retain a response for at most the emitted remaining TTL.

## Validation

Unit tests cover cache hit behavior, ETag/304, path and tag invalidation,
private request bypass, unsafe response bypass, stale-refill prevention, and
oversized body passthrough. A microbenchmark covers a warm cache hit. Multi-
process invalidation and production p95/p99 capacity benchmarks remain required
before claiming issue #71 complete.
