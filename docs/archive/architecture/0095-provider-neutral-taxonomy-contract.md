# ADR-0095: Provider-neutral taxonomy contract

**Status:** Accepted  
**Date:** 2026-09-16

## Context

News and future editorial/commerce editions need categories, tags, localized
labels, and related-content links. Embedding a taxonomy schema in Core or a
specific SQL adapter would violate module isolation and make future providers
hard to support.

## Decision

Define a bounded provider-neutral contract in `internal/taxonomy`. Terms have
an explicit kind, stable slug, localized labels, and an optional parent. Content
relations reference term IDs without exposing storage details. Validation rejects
duplicates, unknown parents, cycles, unsafe identifiers, unknown locales, and
unbounded relation sizes. Persistence, indexing, and module-specific UI remain
explicit module work.

## Consequences

- Modules can adopt the same contract without sharing SQL tables.
- Compact profiles incur no taxonomy resources unless a module enables them.
- Deterministic ordering supports generated manifests and reproducible output.
- Migration/index design and product acceptance are intentionally deferred to
  the adopting module and database contract.

## Verification

`go test ./internal/taxonomy -count=1` covers valid localized relations,
duplicate/unknown/cyclic references, unsafe input, and deterministic ordering.
