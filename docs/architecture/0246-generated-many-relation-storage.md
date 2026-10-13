# ADR-0246: Generated Many-Valued Relations Use Bounded Normalized Storage

- **Status:** Accepted
- **Date:** 2026-09-22
- **Scope:** Generated migrations and repository contracts for `many` relations in Blueprints
- **Related:** ADR-0230, ADR-0235, ADR-0245

## Decision

A `cardinality: many` relation field does not add scalar columns to the source resource table. The generator produces a normalized relation join table with a deterministic name and, for tenant-scoped resources, a composite primary key including `tenant_id`. Listing enforces tenant/source scoping predicates, deterministic ordering, and a `max_items + 1` overflow guard. Replacement methods are atomic within their transactions; `Replace…Tx` variants can participate in broader Edition transaction boundaries. Blueprint limits range from 1 to 100, defaulting to 50.

This does not check foreign key targets for existence or tenant visibility. The Edition must authorize and validate targets within the same transaction. Automated cascade deletion, foreign key constraints, eager loading, and cache invalidation are deliberately omitted from generator scaffolding, remaining under Edition control. The relation table migration is a reviewable proposal applied explicitly through migration registries; runtime DDL is prohibited.

## Consequences

- Schema evolution is additive; rollback reverts application binaries without automatically dropping data tables.
- Common TEXT/INTEGER placeholder conventions are maintained across SQLite and PostgreSQL. New adapters must certify transactional replacement, ordered bounded listing, and tenant predicate enforcement.
- Cache invalidation remains the Edition's responsibility.

## Verification

Generator tests verify normalized DDL and generated API contracts. Client Edition SQLite test fixtures execute migrations, verifying ordered listing within the same tenant, cross-tenant isolation, outer transaction enlistment, and duplicate/overflow rejection.
