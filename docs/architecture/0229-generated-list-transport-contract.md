# ADR-0229: Generated bounded list transport

- **Status:** Accepted
- **Date:** 2026-09-22
- **Related decisions:** [ADR-0225](0225-bounded-resource-query-contract.md), [ADR-0228](0228-public-resource-query-contract.md)

## Decision

Every generated collection repository and service exposes a bounded `List`
operation. The generated HTTP transport supports stable pagination (`limit`,
`page`, `offset`) and an allowlisted sort field with optional descending order.
The repository validates limits again and constructs the order clause only from
generated field names; values remain bound SQL parameters. An empty resource
path is a collection read and still requires the explicit read policy.

This is the first safe API parity slice. Rich field operators and tenant/join
predicates must be bound through the versioned public query AST before they are
enabled by a resource.

## Verification

Generator fixtures compile the List method and transport. Query bounds reject
invalid pages, offsets, limits and sort fields. Editions must add policy tests
for collection reads and dialect-specific index/plan evidence before release.
