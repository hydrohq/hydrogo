# ADR-0200: Reviewed visual collection designer

- Status: Proposed
- Date: 2026-09-21
- Related: ADR-0010, ADR-0133, ADR-0190

## Context

Editors need a visual way to model collections and relationships, but schema
changes affect authorization, migrations, SQLite/PostgreSQL compatibility and
blue/green deployment. Runtime DDL would violate HydroGo's reviewable,
fail-closed product boundary.

## Decision

A future Admin Studio collection designer may produce a versioned collection
specification only. Relations must declare both target collection and `one` or
`many` cardinality. The designer must not apply a migration, create routes, or
enable a module. A reviewed spec continues through the existing generator,
module wiring, dialect migrations and deployment gates.

## Consequences

Non-technical users can prepare a model without hand-writing Go structs, while
an engineer still reviews authorization, migration metadata and rollout. This
does not claim Strapi-style live schema mutation or a generic marketplace.

## Validation

Generator validation rejects relation fields without an explicit target and
bounded cardinality. Future designer browser work requires CSRF, RBAC, export
integrity, SQLite/PostgreSQL generation tests and visual/a11y coverage.
