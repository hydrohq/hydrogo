# ADR-0230: Bounded relation plan contract

- **Status:** Accepted
- **Date:** 2026-09-22
- **Related decisions:** [ADR-0228](0228-public-resource-query-contract.md), [ADR-0210](0210-blueprint-scaffold-boundary.md)

## Decision

Client Editions may expose relation lookups through the public
`extension/v1` contract. Every relation edge declares a provider-neutral target,
cardinality, traversal depth and item budget. The contract caps graph depth,
edge count and related rows before a resolver, picker or eager loader runs.

The contract is a bounded execution plan, not an authorization grant. The
Edition must still apply tenant scope, field policy and target-resource access
inside its resolver. Core never infers joins, cascade behavior or runtime DDL
from relation metadata.

## Verification

Public tests reject invalid cardinality, duplicate edges, excessive depth and
unbounded result budgets. Resource implementations must add SQLite/PostgreSQL
query-plan and N+1 evidence when enabling relation eager loading or async
pickers.
