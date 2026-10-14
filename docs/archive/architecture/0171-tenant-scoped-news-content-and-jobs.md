# ADR-0171: Tenant-scoped news content and scheduled jobs

- Status: Accepted (news repository and payload bridge; job-store isolation open)
- Date: 2026-09-19
- Related decisions: [ADR-0168](0168-host-based-tenant-resolution.md), [ADR-0169](0169-tenant-scoped-worker-composition.md), [ADR-0170](0170-tenant-scoped-pages.md)

## Context

News is consumed by both the public web surface and the future client mobile
API. Scheduled publication is also a background boundary: a scoped repository
without a tenant-bearing job payload would either fail every job or risk
running a tenant A item under tenant B's context.

## Decision

Migration `202609190007_news_tenant_scope` adds a tenant discriminator and
tenant-aware indexes to `news_items`. The news repository scopes lifecycle
writes, route reads, public/admin lists, related content and scheduled-item
discovery when enabled. Scheduled job payloads carry the validated tenant ID;
the worker restores that scope before running the job and dead-letters legacy
payloads without one in shared mode.

## Consequences

- Web/mobile news reads and editorial transitions cannot cross tenant IDs via
  an item ID or route.
- Existing news rows require controlled tenant backfill before shared mode.
- The generic jobs table is still a shared queue; tenant payload validation and
  future queue-level tenant predicates remain required for full certification.
