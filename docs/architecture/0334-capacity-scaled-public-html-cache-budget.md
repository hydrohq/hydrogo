# ADR-0334: Capacity-profile-scaled public HTML cache budget

- **Status:** Accepted (empirical thresholds; production soak open)
- **Date:** 2026-09-30
- **Related:** [ADR-0329](0329-bounded-public-html-l1.md) (partially superseded),
  [ADR-0190](0190-capacity-profiles-concurrency-budgets.md), [ADR-0333](0333-http-admission-concurrency-limit.md)

## Context

ADR-0329's bounded public HTML cache used one fixed size for every
deployment: 128 entries, 32 MiB total body data, 512 KiB per body. This was
never wired to `internal/capacity.Budget` — `cmd/hydro/main.go` constructed it
with `server.PublicHTMLCacheOptions{}` (all zero), so every profile
(Compact, Standard, Distributed) silently fell back to the same fixed
defaults, unlike every other bounded resource in ADR-0190 (DB pool size,
worker concurrency, queue depth, HTTP body limit), which already scale by
profile.

Live measurement on the 2-vCPU / 908-MiB benchmark host during the
2026-09-30 capacity work
(`docs/validation/hydrogo-capacity-benchmark-2026-09-30.md`) showed the
HydroGo process itself using well under 200 MiB RSS even under sustained
320-client load, with 360–400+ MiB of host memory free throughout. CPU, not
RAM, was the limiting resource on this host. A larger HTML cache trades spare
RAM for fewer expensive re-renders (fewer cache misses under real traffic
with more distinct pages than a single-route synthetic benchmark exercises),
which is a direct RAM-for-CPU trade rather than a change to what the cache
is allowed to store or how it invalidates.

## Decision

Add `PublicHTMLCacheEntries`, `PublicHTMLCacheMaxBytes`, and
`PublicHTMLCacheMaxBodyBytes` to `capacity.Budget`
(`internal/capacity/profile.go`):

| Profile | Entries | Total bytes | Per-body limit |
|---|---:|---:|---:|
| Compact | 128 | 16 MiB | 512 KiB |
| Standard | 512 | 128 MiB | 1 MiB |
| Distributed | 2,048 | 512 MiB | 2 MiB |

Compact keeps a size close to ADR-0329's original fixed numbers (even
slightly smaller on total bytes), preserving the "unused capability imposes
no burden" rule for the smallest profile. Standard and Distributed scale up
because they are assumed to run on hosts with materially more spare memory,
which the 2026-09-30 benchmark host — nominally a Standard-class box —
already demonstrated it had.

`cmd/hydro/main.go` now constructs `server.NewPublicHTMLCache` from these
budget fields instead of an empty `PublicHTMLCacheOptions{}`. The outer safety
ceilings in `internal/server/public_html_cache.go`
(`publicHTMLEntryLimit`, `publicHTMLByteLimit`, `publicHTMLBodyLimit`) were
raised from 128/32 MiB/512 KiB to 4,096/512 MiB/2 MiB so no profile's budget
is silently clamped back down to the old fixed size; they remain a defensive
outer bound independent of what any given profile requests.

These numbers are an initial empirical estimate sized from the one
benchmark host described above, not a hardware-independent guarantee — the
same caveat ADR-0333 carries for its admission thresholds. They should be
revisited once measured against a real multi-route production workload,
where cache pressure (many distinct pages, not one synthetic route) behaves
differently than this benchmark's single hot route.

## Database and migration impact

None. No schema, migration, or external service is introduced.

## Cache and capacity impact

This changes only the *size* of the bound, not the cache's behavior: the
same TTL, invalidation-generation tracking, singleflight coalescing
(ADR-0329, 2026-09-30 update), and allowlist rules apply unchanged at every
size. A larger cache means a process that has been running for a while under
varied traffic can hold more distinct pages, which is the intended effect;
it also means a full cache eviction sweep touches more entries, though this
remains bounded and infrequent (LRU-like oldest-expiry eviction, as
ADR-0329 already specified).

Memory accounting for capacity planning changes proportionally: ADR-0329
noted that deployment capacity planning multiplies the per-process cache
ceiling by the number of web processes. With the Standard profile's new
128 MiB ceiling (versus the old fixed 32 MiB), that multiplication factor is
four times larger; operators sizing a multi-process Standard deployment
should account for this.

## Deployment and compatibility

Built into the existing binary; no runtime dependency added. Safe across
Blue/Green: each process generation applies its own configured budget
independently, and a process restart clears the cache regardless of its
configured size, matching ADR-0329's existing behavior.

## Validation

`internal/capacity` unit tests cover `Budget.Validate()` rejecting a
zero/negative value for any of the three new fields. The existing
`internal/server/public_html_cache_test.go` suite (cache hit/invalidation,
CSP nonce regeneration, singleflight coalescing, body-limit passthrough)
exercises the cache's behavior independent of its configured size and
continues to pass with the raised outer ceilings. The specific per-profile
numbers above are not yet certified by a production soak against a
multi-route workload; that remains open, matching ADR-0333's qualification.
