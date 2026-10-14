# ADR-0128: Bounded memory search provider

- **Status:** Accepted
- **Date:** 2026-09-16
- **Related decisions:** [ADR-0123](0123-versioned-search-provider-contract.md), [ADR-0010](0010-sdk-lifecycle-and-edition-manifest.md)

## Decision

HydroGo ships a provider-neutral `internal/search.MemoryProvider` for Compact
and prototype profiles. It keeps a bounded in-memory projection, supports
replace-all rebuilds plus upsert/delete, validates locale and document identity,
and returns deterministic score/ID ordering with bounded pagination. Modules
remain responsible for indexing only published and authorized projections.

The provider is not a Distributed production search guarantee. External search
engines, durable indexing, rebuild jobs, freshness metrics and capacity evidence
remain explicit adapter/deployment work.

## Evidence and consequences

`internal/search/memory_test.go` covers ranking, locale filtering, pagination,
rebuild replacement, deletion, cancellation and the document-count bound. The
adapter has no database, cache or external service dependency, so inactive
modules do not incur resource overhead.
