# ADR-0176: Tenant-scoped form registry

- Status: Accepted (form schema repository boundary; submission/backfill operations remain open)
- Date: 2026-09-19
- Related decisions: [ADR-0172](0172-tenant-scoped-site-data.md), [ADR-0168](0168-host-based-tenant-resolution.md), [ADR-0175](0175-tenant-scoped-commerce-checkout.md)

## Context

The form registry is used by site and admin workflows to load validated form
schemas. In a shared deployment, an ID-only schema lookup lets one tenant reuse
or delete another tenant's custom form definition.

## Decision

Migration `202609190012_forms_tenant_scope` adds a tenant discriminator and
index to `hydro_forms`. The SQL store requires a validated tenant context in
shared mode for upsert, get, list and delete, while preserving the unscoped
single-tenant compatibility path. Existing forms remain in the empty namespace
until controlled backfill and provisioning are available.

## Consequences

- Form schema access fails closed without tenant context and cannot cross tenant
  IDs.
- Form submission inbox isolation remains owned by the tenant-scoped Site
  repository; end-to-end submission routing and backfill are still release gates.
