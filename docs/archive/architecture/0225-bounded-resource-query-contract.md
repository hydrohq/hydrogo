# ADR-0225: Bounded resource query contract

- **Status:** Accepted
- **Date:** 2026-09-22
- **Related decisions:** [ADR-0006](0006-extensible-database-adapters.md), [ADR-0114](0114-generated-query-input-bounds.md), [ADR-0140](0140-tenant-isolation-scope.md)

## Decision

Generated resource transports may accept Payload-style query parameters, but
they must be parsed into HydroGo's bounded `internal/query` AST before any
provider-specific statement is built. Field names and operators come from an
Edition-owned allowlist; values are always bound arguments. The contract
supports bounded equality, pattern, range, existence, `in`, `and`/`or`, sort,
limit and offset controls.

The parser enforces maximum condition/value/sort/page budgets. SQL rendering
supports SQLite and PostgreSQL placeholders without exposing dialect details to
domain or service code. Tenant predicates and authorization filters are added
by the application service/policy layer, not by untrusted query parameters.
The parser and SQL builder apply the same bounds, and the builder revalidates
directly constructed ASTs at the SQL boundary; see [ADR-0256](0256-resource-query-ast-validation-budgets.md).

## Consequences

- Deep querying is ergonomic without accepting arbitrary SQL or columns.
- Query complexity and pagination are predictable for Compact deployments.
- A future operator or relation traversal requires an explicit contract and
  corresponding dialect/authorization tests.

## Verification

`internal/query` tests cover parameter binding, unknown field/operator denial,
value and sort budgets, OR grouping, and SQLite/PostgreSQL placeholder output.
