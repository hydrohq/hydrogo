# ADR-0180: PostgreSQL tenant RLS enforcement at transaction boundaries

- Status: Accepted (defense-in-depth phase; direct pool call-site audit remains open)
- Date: 2026-09-20
- Supersedes: ADR-0156 foundation-only binding

## Context

The repository and service contracts already require an explicit tenant scope
in shared mode, but PostgreSQL did not yet enforce that boundary below the
application predicate. A missed predicate, future repository, or accidental
cross-tenant write could therefore bypass the intended isolation contract.

## Decision

The tenant migration enables and forces row-level security on every table that
has a `tenant_id` discriminator. The policy compares `tenant_id` with the
transaction-local `app.current_tenant_id` setting only when
`app.tenant_rls_enforced` is `true`. `database.NewTenantTransactor` sets both
settings before the callback: shared mode requires `tenant.FromContext`, while
single-tenant mode explicitly sets enforcement to `false` for compatibility.

The policy deliberately permits an unset enforcement setting for legacy direct
pool reads during the migration window. Shared production wiring must use the
tenant transactor for every tenant-scoped operation; direct `*sql.DB` call
sites are an explicit follow-up audit and are not considered RLS-certified by
this ADR.

## Consequences

- A shared transaction with a missing or mismatched tenant setting cannot read,
  insert, or update another tenant's rows.
- Existing single-tenant PostgreSQL installations continue to work when their
  transactions use the explicit single-tenant binding.
- RLS is defense in depth, not a replacement for repository predicates,
  authorization, audit records, or tenant provisioning/backfill controls.
- The migration is forward-only; rollback is restore or a follow-up migration.

## Verification

- `internal/tenant` tests verify shared and single-tenant GUC binding.
- `internal/database` and `cmd/hydro` tests verify the new transactor wiring
  compiles and existing SQLite behavior remains intact.
- A live PostgreSQL shared/single-tenant isolation drill is still required
  before the shared SaaS profile can be certified.
