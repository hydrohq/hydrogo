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

The transport now binds the versioned public query AST and generated field
allowlist, so equals/contains/in/exists/range operators are converted to
prepared SQL arguments before repository execution. Tenant predicates and
join-depth budgets remain explicit Edition policy and relation contracts.

## Verification

Generator fixtures compile the List method and transport. Query bounds reject
invalid pages, offsets, limits and sort fields. Editions must add policy tests
for collection reads and dialect-specific index/plan evidence before release.
