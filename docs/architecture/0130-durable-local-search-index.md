# ADR-0130: Bounded durable local search index

- **Status:** Accepted
- **Date:** 2026-09-16
- **Related decisions:** [ADR-0123](0123-versioned-search-contract.md), [ADR-0128](0128-bounded-memory-search-provider.md), [ADR-0033](0033-runtime-module-resource-gating.md)

## Decision

The optional `search` capability may enable a provider-neutral SQL projection
named `hydro_search_documents`. The projection is bounded and stores only
module-owned, already-authorized searchable fields. `SQLProvider` supports
SQLite and PostgreSQL with parameterized statements, deterministic ordering,
bounded reads, cancellation and atomic rebuilds when called with a transaction.
The migration is module-scoped, so Compact deployments without search do not
create the table or index.

The memory provider remains the default for prototypes. Full-text engines and
external indexing are separate adapters and remain environment-gated; they
must not leak provider-specific behavior into module services.

## Evidence and limits

`internal/search/sql_test.go` covers rebuild, locale filtering, pagination,
deletion, duplicate rejection and index capacity on SQLite. The migration
contains equivalent SQLite/PostgreSQL DDL and checksum metadata. Production
query-shape, freshness, failover and capacity evidence for an external search
engine is still required before a distributed profile claim.
