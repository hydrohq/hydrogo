# ADR-0170: Tenant-scoped page content

- Status: Accepted (page repository boundary; backfill and other content modules open)
- Date: 2026-09-19
- Related decisions: [ADR-0154](0154-tenant-scope-primitive.md), [ADR-0168](0168-host-based-tenant-resolution.md), [ADR-0169](0169-tenant-scoped-worker-composition.md)

## Context

Pages are the primary CMS data surface. Resolving a tenant at the HTTP layer
does not provide isolation if page repository queries still select by ID or
route alone. A shared page slug, admin list, draft update or publish operation
could otherwise cross tenant boundaries.

## Decision

Migration `202609190006_pages_tenant_scope` adds tenant discriminators to
`pages` and `page_revisions`, changes route uniqueness and list indexes to be
tenant-aware, and preserves empty discriminators for existing single-customer
rows. The page SQL store uses the validated context scope for all reads,
updates, publishes and revision writes when `RequireTenantScope` is enabled;
missing scope fails closed.

## Consequences

- Public routes, Studio lists and draft/publish workflows cannot cross tenants
  through page IDs or slugs in shared mode.
- Existing rows require an operator-controlled tenant backfill before shared
  mode is enabled.
- News, site, commerce and member repositories still require their own scoped
  migrations and call-site certification.
