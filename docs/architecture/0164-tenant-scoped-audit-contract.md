# ADR-0164: Tenant-scoped audit contract

- Status: Accepted (storage primitive; query and operator workflow propagation open)
- Date: 2026-09-19
- Related decisions: [ADR-0162](0162-tenant-scoped-outbox-contract.md), [ADR-0154](0154-tenant-scope-primitive.md)

## Context

Audit data is itself tenant data. A shared audit table that records actions
without a tenant discriminator can leak history through operator views or make
erasure/export boundaries impossible to enforce.

## Decision

Migration `202609190004_audit_tenant_scope` adds a `tenant_id` discriminator and
lookup index to SQLite and PostgreSQL. The audit SQL store automatically selects
tenant-aware writes when a validated context scope exists, provides
`RecordScoped`, and supports explicit `RequireTenantScope` fail-closed mode.
Request correlation remains intact.

## Consequences

- New shared-mode audit writes have a storage-level tenant identity.
- Existing single-customer audit rows remain readable with the default-empty
  discriminator during migration.
- Audit listing/export, retention, erasure, super-admin access and all module
  composition roots still require tenant-aware implementation and certification.
