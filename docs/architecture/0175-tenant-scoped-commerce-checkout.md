# ADR-0175: Tenant-scoped commerce checkout and payments

- Status: Superseded by ADR-0199
- Date: 2026-09-19
- Related decisions: [ADR-0174](0174-tenant-scoped-commerce-catalog.md), [ADR-0173](0173-tenant-scoped-members-and-sessions.md), [ADR-0168](0168-host-based-tenant-resolution.md)

## Context

Catalog isolation alone is insufficient for a mobile commerce client. Orders,
carts, order lines, payment transactions, refund claims and webhook receipts
contain customer and financial state. An ID-only lookup or worker mutation can
otherwise disclose or mutate another tenant's checkout.

## Decision

Migration `202609190011_commerce_checkout_tenant_scope` adds tenant
discriminators and lookup indexes to checkout and payment tables. In shared
mode the SQL store requires a validated tenant context on order, cart, payment,
refund and expiry operations. Order-line and product/variant joins are
tenant-predicated, and the member customer-order adapter uses the same scoped
store. Existing records remain in the empty namespace pending controlled
backfill and tenant provisioning.

## Consequences

- Cross-tenant order, cart and payment IDs fail as not found; missing scope
  fails closed.
- Webhook receipt claiming is tenant-bound; provider event IDs remain globally
  unique for compatibility.
- Global user/idempotency/SKU uniqueness is retained until identity and
  provisioning contracts are made tenant-aware.
- PostgreSQL RLS, production webhook routing and backfill drills remain release
  gates outside this repository boundary.
