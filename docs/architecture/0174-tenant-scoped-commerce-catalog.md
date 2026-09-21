# ADR-0174: Tenant-scoped commerce catalog

- Status: Superseded by ADR-0199
- Date: 2026-09-19
- Related decisions: [ADR-0168](0168-host-based-tenant-resolution.md), [ADR-0172](0172-tenant-scoped-site-data.md), [ADR-0173](0173-tenant-scoped-members-and-sessions.md)

## Context

The client/mobile catalog must never expose a product or variant belonging to a
different host-resolved tenant. Product and variant stock mutations are also
tenant-owned state, so an ID-only update is an isolation boundary violation.

## Decision

Migration `202609190010_commerce_catalog_tenant_scope` adds tenant
discriminators and lookup indexes to products and variants. In shared mode the
commerce catalog store requires a validated tenant context for create, read,
list and stock-reservation operations, and predicates both sides of the
product/variant join. Existing rows remain in the empty namespace until an
explicit backfill and provisioning workflow is delivered.

## Consequences

- Web and mobile catalog reads fail closed without tenant context and cannot
  cross tenant IDs.
- Single-tenant deployments retain the legacy unscoped path.
- Orders, carts, payments, refunds and customer-order readers still need a
  separate tenant contract before shared commerce is release-ready.
- Global SKU uniqueness remains a compatibility constraint until tenant-aware
  identity provisioning is approved.
