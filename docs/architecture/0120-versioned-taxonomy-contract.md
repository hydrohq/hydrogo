# ADR-0120: Versioned taxonomy contract

- **Status:** Accepted
- **Date:** 2026-09-16
- **Related decisions:** [ADR-0010](0010-sdk-lifecycle-and-edition-manifest.md), [ADR-0012](0012-content-media-and-portability.md)

## Decision

Localized categories, tags, parent relationships and content relations are
exposed as a provider-neutral `extension/v1` contract. Modules own relational
tables, indexes, migrations and authorization; the SDK contract does not expose
SQL, cache or provider details. Validation is bounded and deterministic so
generated Client Editions can reject malformed taxonomy before persistence.

The existing `internal/taxonomy` package remains Core-only compatibility code.
Relational persistence and admin editing stay module-owned.

## Evidence and consequences

`extension/v1/taxonomy_test.go` proves localized terms, parent-cycle rejection,
unknown relation rejection and deterministic copies. This closes the SDK
contract portion; relational persistence, admin editing and multilingual UX
remain product work and stay in the open-gates list.
