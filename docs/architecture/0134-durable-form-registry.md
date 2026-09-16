# ADR-0134: Durable bounded form registry

- **Status:** Accepted
- **Date:** 2026-09-16
- **Related decisions:** [ADR-0121](0121-versioned-forms-contract.md), [ADR-0133](0133-schema-driven-form-builder-primitive.md), [ADR-0033](0033-runtime-module-resource-gating.md)

## Decision

The optional `forms` capability stores validated provider-neutral form schemas
as bounded JSON rows in `hydro_forms`. The registry supports upsert, get, list
and delete for at most 1000 forms. SQLite and PostgreSQL use equivalent
constraints and parameterized statements. Stored JSON is validated again on
read, so corruption fails closed rather than reaching a submission handler.

The migration is module-scoped and does not add tables to profiles without
forms. Authorization, publication, submission retention and Admin route
composition remain Site/Client Edition responsibilities.

## Evidence

`internal/forms/sqlstore/store_test.go` covers round-trip, listing, deletion,
missing-record behavior and invalid schema rejection on SQLite. Admin Studio’s
schema editor can now target this registry without inventing another contract.
