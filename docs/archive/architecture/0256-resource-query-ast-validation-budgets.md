# ADR-0256: Revalidate bounded resource query ASTs

- **Status:** Accepted
- **Date:** 2026-09-23
- **Related decisions:** [ADR-0225](0225-bounded-resource-query-contract.md), [ADR-0228](0228-public-resource-query-contract.md), [ADR-0235](0235-generated-tenant-scoped-resource-contract.md)

## Context

The query parsers enforced several bounds, but `BuildSQL` is also callable with
directly constructed public AST values. In addition, repeated scalar values,
ambiguous pagination controls, permissive `exists` values, and unbounded `in`
expansion could produce ambiguous or oversized SQL plans. Parser and builder
validation must agree because the builder is the last boundary before SQL is
returned to an Edition repository.

## Decision

Both `internal/query` and public `extension/v1` query contracts enforce the
same budgets: 32 conditions, 256-byte values, 32 members per `in`, 256 total
bound arguments, 64 query keys, 16 KiB aggregate key/value input, 8 sort
fields, limit 1–100, and offset 0–10,000. `page` and `offset` cannot be mixed.
Scalar operators accept exactly one value; `exists` accepts exactly `true` or
`false`. Empty `in` members are rejected.

`BuildSQL` independently revalidates query and sort identifiers against the
resource allowlist, field-name consistency, group names, value cardinality and
length, total argument count, sort count, and pagination. SQL values remain
bound parameters. This decision tightens ADR-0225/0228 without changing their
operator or grouping semantics and adds no database schema, runtime service,
or dependency.

## Consequences

- Direct AST callers receive the same abuse bounds as URL parser callers.
- Large `in` lists fail deterministically before database-driver parameter
  limits or excessive SQL allocation are reached.
- Editions using duplicate scalar query values or both `page` and `offset`
  must normalize requests before invoking the contract.

## Verification

Internal and public contract tests cover parser bounds, strict `exists`,
scalar cardinality, pagination conflicts, direct AST validation, and argument
budgets. Full SQLite/PostgreSQL query contract suites remain part of the
repository test gate.
