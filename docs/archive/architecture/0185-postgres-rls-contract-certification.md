# ADR-0185: PostgreSQL tenant RLS contract isolation certification

- Status: Accepted
- Date: 2026-09-20
- Related decisions: [ADR-0156](0156-postgres-tenant-rls-binding.md), [ADR-0180](0180-postgres-tenant-rls-enforcement.md), [ADR-0181](0181-auth-identity-rls-call-sites.md), [ADR-0182](0182-member-store-rls-call-sites.md), [ADR-0183](0183-forms-media-rls-call-sites.md), [ADR-0184](0184-mobile-store-tenant-rls-bindings.md)

## Context

HydroGo multi-tenant SaaS requires certified database-level isolation on PostgreSQL. While application services filter by tenant ID predicates, defense-in-depth requires PostgreSQL Row-Level Security (RLS) to enforce that even if an application predicate were missing or corrupted, no cross-tenant read, insert, update, or delete can occur.

## Decision

We establish an end-to-end PostgreSQL RLS contract isolation suite in `tests/contract/postgres_rls_contract_test.go` certifying all 29 tenant-scoped PostgreSQL tables:
1. `ENABLE ROW LEVEL SECURITY` and `FORCE ROW LEVEL SECURITY` are active across every table with a `tenant_id` column.
2. The `hydrogo_tenant_isolation` policy exists and evaluates `tenant_id = nullif(current_setting('app.current_tenant_id', true), '')` when `app.tenant_rls_enforced = 'true'`.
3. Strict isolation tests verify that transactions scoped to `tenant-alpha` cannot read, update, or delete `tenant-alpha` records when executing under `tenant-beta` context, even when application `WHERE tenant_id` predicates are omitted.
4. `WITH CHECK` constraints prevent inserting records with spoofed tenant IDs.
5. Missing or unconfigured tenant contexts fail closed.

## Consequences

- Automated regression and contract tests verify complete RLS enforcement across all tables.
- Multi-tenant shared PostgreSQL deployments are formally certified for row-level tenant boundary isolation.
