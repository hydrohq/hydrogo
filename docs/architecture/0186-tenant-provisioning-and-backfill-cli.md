# ADR-0186: Tenant provisioning and legacy data backfill CLI

- Status: Superseded by ADR-0201 for tenant registry, idempotent provisioning, and active-tenant bootstrap backfill
- Date: 2026-09-20
- Related decisions: [ADR-0149](0149-shared-multi-tenant-saas-evolution.md), [ADR-0154](0154-tenant-scope-primitive.md), [ADR-0168](0168-host-based-tenant-resolution.md), [ADR-0180](0180-postgres-tenant-rls-enforcement.md), [ADR-0185](0185-postgres-rls-contract-certification.md)

## Context

Transitioning single-tenant or legacy installations of HydroGo to multi-tenant shared SaaS requires operational tooling to:
1. Validate tenant host routing and domain configuration ahead of deployment.
2. Safely audit and backfill existing unassigned records (`tenant_id = ''`) across the historical tenant-scoped tables to a designated initial tenant ID.
3. Query active tenants and ensure database consistency without leaking dialect details or internal structures.

## Decision

We introduce the `hydro tenant` CLI command namespace:
1. `hydro tenant validate`: Validates operator host-to-tenant configuration mappings, syntax rules, and fail-closed readiness. Supports human-readable and `--json` machine-readable output.
2. `hydro tenant backfill --tenant-id <id> [--dry-run|--confirm]`:
   - `--dry-run`: Plans backfill across the historical tenant-scoped tables, returning counts of unassigned records per table without modifying the database.
   - `--confirm`: Atomically updates all unassigned records to the target tenant ID within a single database transaction. On PostgreSQL, temporarily bypasses transaction RLS enforcement for the migration boundary.
3. `hydro tenant list`: Enumerates distinct tenant IDs across database tables and configured host mappings.

## Consequences

- Operators can safely migrate existing single-tenant databases into shared multi-tenant SaaS environments with zero data corruption.
- Dry-run verification enables pre-flight checks and audit compliance before committing changes.
