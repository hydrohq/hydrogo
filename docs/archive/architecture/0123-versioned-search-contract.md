# ADR-0123: Versioned search provider contract

- **Status:** Accepted
- **Date:** 2026-09-16
- **Related decisions:** [ADR-0012](0012-content-media-and-portability.md), [ADR-0010](0010-sdk-lifecycle-and-edition-manifest.md)

## Decision

Optional search providers implement the bounded `extension/v1` contract. Query
length, locale, limit and offset are validated before provider calls. Results
are copied, hard-bounded and sorted deterministically by score then ID. Search
is a derived projection: modules must re-check authorization and publication
state before exposing hits, and no external index becomes the content source of
truth.

## Evidence and consequences

`extension/v1/search_test.go` covers query bounds, locale validation and stable
hard-bounded results. Provider-specific indexing, rebuild, freshness and
capacity evidence remain module/deployment concerns.
