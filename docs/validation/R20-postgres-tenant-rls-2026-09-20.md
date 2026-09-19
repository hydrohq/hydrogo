# R20 — PostgreSQL tenant RLS transaction binding

## Scope

This phase adds the PostgreSQL RLS migration for every current tenant-scoped
table and binds the tenant GUCs from the shared-mode transaction constructor.
It does not claim the live PostgreSQL isolation drill or certify direct pool
call sites.

## Evidence

- Migration: `202609200003_postgres_tenant_rls`
- ADR: [0180](../architecture/0180-postgres-tenant-rls-enforcement.md)
- Shared binding sets `app.tenant_rls_enforced=true` and
  `app.current_tenant_id` transaction-locally.
- Single-tenant binding sets `app.tenant_rls_enforced=false` transaction-locally.
- `go test ./internal/tenant ./internal/database ./cmd/hydro` passed on
  Windows with the repository toolchain.

## Limitations and next gate

The policy preserves compatibility for legacy direct `*sql.DB` operations when
the enforcement GUC is unset. The next phase must inventory and convert every
shared-mode direct pool call site, then run a real PostgreSQL drill proving
same-tenant access, cross-tenant read denial, cross-tenant write denial, and
missing-binding denial.
