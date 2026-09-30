# ADR-0333: Bounded HTTP admission concurrency limit

- **Status:** Accepted (empirical thresholds; production soak open)
- **Date:** 2026-09-30
- **Related:** [ADR-0190](0190-capacity-profiles-concurrency-budgets.md), [ADR-0329](0329-bounded-public-html-l1.md)

## Context

ADR-0190 bounds database pools, worker concurrency, queue depth, and body size
per capacity profile, but it does not bound how many HTTP requests a process
admits concurrently. The 2026-09-30 capacity benchmark
(`docs/validation/hydrogo-capacity-benchmark-2026-09-30.md`) showed that once
CPU saturates, additional clients only add queueing delay: throughput plateaus
around 800 req/s (warm cache) and 500 req/s (dynamic) on the tested 2-vCPU
host, while p95/p99 keep climbing until requests time out at 1,000 clients. A
process that keeps accepting work it cannot serve promptly slows down the
requests it already admitted.

## Decision

Add `Budget.MaxInFlightHTTP` to the capacity profile (`internal/capacity/profile.go`):
Compact=32, Standard=150, Distributed=600. These are initial empirical
estimates sized to the observed single-host saturation points, not a
hardware-independent guarantee; they must be revisited once a production
soak or a larger host class is measured.

`server.ConcurrencyLimitMiddleware` (`internal/server/concurrency_limit.go`)
enforces this budget as the outermost middleware in the handler chain. A
request beyond the cap fails immediately with `503` and a `Retry-After: 1`
header instead of queuing behind already-admitted work — a fail-fast strategy
so CPU already committed to in-flight requests is not starved by requests the
process cannot serve anyway. A non-positive limit (not expected in a valid
profile, but honored defensively) disables the middleware with zero added
overhead, preserving the Compact-profile zero-footprint rule.

This budget is process-local, like the ADR-0329 HTML cache: it does not
coordinate admission across multiple processes or nodes. A Distributed
deployment's real capacity is the sum of per-process limits behind the load
balancer, not a cluster-wide guarantee.

## Database and migration impact

None. No schema, migration, or external service is introduced.

## Cache and capacity impact

Interacts with, but does not replace, ADR-0329's cache. This ADR only bounds
how many requests — cached or not — a process serves at once. Separately,
concurrent cache misses on the same route are now coalesced with
`golang.org/x/sync/singleflight` (ADR-0329 update, 2026-09-30): each waiting
request still renders its own CSP nonce from the shared entry, so nothing is
replayed between distinct client responses.

## Deployment and compatibility

Built into the existing binary; no runtime dependency added. Safe across
Blue/Green: old and new process generations each enforce their own limit
independently. Operators observe shed requests as `503` responses with
`Retry-After` in the existing HTTP error/metrics middleware; no new
observability surface is introduced by this ADR.

## Validation

Unit tests (`internal/server/concurrency_limit_test.go`) cover pass-through
when disabled and 503 shedding with `Retry-After` at capacity. The default
thresholds themselves are not yet certified by a production soak; that
remains open, matching the qualification already carried by ADR-0190's
profile numbers.
