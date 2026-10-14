# ADR-0228: Public bounded resource query contract

- **Status:** Accepted
- **Date:** 2026-09-22
- **Related decisions:** [ADR-0225](0225-bounded-resource-query-contract.md), [ADR-0210](0210-blueprint-scaffold-boundary.md)

## Decision

Generated Client Editions consume the bounded resource-query parser from the
public `extension/v1` contract. They must not import HydroGo `internal`
packages. The contract allowlists fields and operators supplied by each
resource, caps conditions, values, sorting, page size and offset, and returns a
provider-neutral AST. SQL dialect adaptation remains an Edition repository
concern.

This keeps Core's dependency boundary honest while preserving the same safety
budgets as the internal query implementation. A future major contract version
may add operators only with compatibility and abuse-budget review.

## Verification

Public contract tests cover allowlisted operators, pagination, bounds and
prepared SQLite/PostgreSQL placeholder generation. Generated transports bind
this parser before constructing repository queries. Editions record
dialect-specific SQL plans and index behavior in their resource review
evidence. The parser and SQL builder budgets are aligned and independently
enforced at the builder boundary as documented in [ADR-0256](0256-resource-query-ast-validation-budgets.md).
