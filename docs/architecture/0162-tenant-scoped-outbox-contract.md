# ADR-0162: Tenant-scoped outbox contract

- Status: Accepted (storage primitive; end-to-end propagation open)
- Date: 2026-09-19
- Related decisions: [ADR-0154](0154-tenant-scope-primitive.md), [ADR-0156](0156-postgres-tenant-rls-binding.md), [ADR-0158](0158-bounded-outbox-polling.md)

## Context

The outbox is a background-job trust boundary. A shared database cannot rely on
aggregate IDs or worker selection alone to prevent one tenant's event from being
claimed or acknowledged by another tenant.

## Decision

Migration `202609190003_outbox_tenant_scope` adds a `tenant_id` discriminator
and claim index to SQLite and PostgreSQL outbox tables. The SQL store exposes
`EnqueueScoped`, `ClaimScoped`, `AckScoped`, and `RetryScoped`; each requires a
validated `tenant.Scope` in context and includes the tenant predicate in the
read/write operation. Legacy methods remain for the current single-customer
profile and write the default-empty discriminator.

## Consequences

- Scoped callers get a fail-closed storage API and cannot lease another
  tenant's event through the store boundary.
- Existing single-customer deployments remain readable during migration.
- All module/application enqueue call sites, worker lifecycle, RLS policies,
  provisioning and tenant-aware webhook delivery still must be migrated before
  shared tenancy is enabled.
