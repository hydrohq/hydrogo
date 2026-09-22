# ADR-0258: Enforce Blueprint uniqueness at the database boundary

- **Status:** Accepted
- **Date:** 2026-09-23
- **Related decisions:** [ADR-0226](0226-versioned-field-validation-contract.md), [ADR-0235](0235-generated-tenant-scoped-resource-contract.md), [ADR-0246](0246-generated-many-relation-storage.md)

## Context

Blueprint `validation.unique` was preserved as metadata but did not generate a
constraint. Enforcing it with an application-side preflight query would race
under concurrent writes and could leak records across tenant boundaries.

## Decision

For each scalar field marked `unique`, the generator proposes a deterministic
unique index in the reviewable migration. Non-tenant resources use
`(field_name)`; tenant-scoped resources use `(tenant_id, field_name)`, making
the default scope tenant-local. Many-valued relation fields cannot declare
scalar uniqueness; one-valued relations may.

SQLite and PostgreSQL both enforce the same index semantics. Nullable unique
fields inherit each provider's standard behavior, allowing multiple NULLs;
empty strings are values and therefore conflict when repeated within scope.
The generator warns operators to detect/resolve duplicate existing values
before applying the migration. It emits no runtime DDL or duplicate preflight
query.

Repository/driver errors remain unchanged; stable API error classification is
not added by this decision. Editions must translate recognized uniqueness
violations to a user-safe conflict response without exposing driver messages.

## Consequences

- Concurrent writes cannot bypass declared uniqueness once the migration is
  applied.
- Tenant-local uniqueness is explicit and avoids cross-tenant information
  coupling.
- Adding `unique: true` to an already-published resource changes its generated
  migration and requires the existing compatibility process, duplicate audit,
  and operator-approved cleanup plan.
- Uniqueness does not define immutability or defaults. Immutable scalar
  enforcement is separately defined by [ADR-0260](0260-generated-immutable-field-enforcement.md);
  create defaults still require field-presence semantics distinct from a zero
  value and remain open.

## Verification

Generator tests execute generated migrations against SQLite and verify global
and per-tenant uniqueness behavior. Migration SQL uses only portable index
syntax; live PostgreSQL execution remains an Edition/release certification
gate.
