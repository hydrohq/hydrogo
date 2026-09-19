# ADR-0165: Tenant-scoped media metadata

- Status: Accepted (metadata boundary; object namespace and read workflows open)
- Date: 2026-09-19
- Related decisions: [ADR-0154](0154-tenant-scope-primitive.md), [ADR-0163](0163-tenant-context-outbox-propagation.md)

## Context

Media IDs and storage metadata are tenant-owned data. Filtering only content
tables while leaving media reads, visibility transitions, or derivative records
global creates an IDOR and cross-tenant object disclosure path.

## Decision

Migration `202609190005_media_tenant_scope` adds `tenant_id` to
`media_assets` and `media_variants` in SQLite and PostgreSQL. The media SQL
store automatically applies a validated context scope to metadata reads,
listing, upload state transitions and derivative operations. `RequireTenantScope`
rejects missing scope in shared mode; legacy single-customer contexts retain
the old path during migration.

## Consequences

- Scoped media metadata cannot be read or state-transitioned through another
  tenant's ID.
- Object-store key namespace, signed URL policy, HTTP authorization and media
  export/erasure still need explicit tenant propagation before shared launch.
- Existing default-empty rows require a controlled backfill/tenant assignment
  before enabling shared mode.
