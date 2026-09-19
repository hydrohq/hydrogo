# ADR-0155: Tenant-scoped cache API

- Status: Accepted (opt-in foundation; shared multi-tenant mode disabled)
- Date: 2026-09-19
- Related decisions: [ADR-0140](0140-tenant-isolation-scope.md), [ADR-0154](0154-tenant-scope-primitive.md)

## Decision

The bounded local cache and optional two-tier cache expose
`GetOrLoadScoped(ctx, module, key, loader)`. The method requires an explicit
validated tenant scope and applies the deterministic tenant prefix to both L1
and L2 keys before any lookup or load. Missing scope or malformed components
fail closed through `internal/tenant`.

The existing unscoped methods remain for public/single-tenant projections and
are not silently treated as tenant-safe. Callers enabling shared tenancy must
use the scoped API until repository, media, jobs, audit, export/restore and RLS
propagation are separately certified.

## Consequences

- Shared-process L1 cache entries and distributed L2 entries cannot collide
  across validated tenant scopes when the scoped API is used.
- Compact/single-tenant callers pay no tenant dependency or key overhead unless
  they opt into the scoped method.
- This does not certify end-to-end tenant isolation; unscoped call-site
  inventory and PostgreSQL RLS remain required release gates.
