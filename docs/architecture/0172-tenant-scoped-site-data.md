# ADR-0172: Tenant-scoped site data

- Status: Accepted (site repository boundary; forms module and operator workflows open)
- Date: 2026-09-19
- Related decisions: [ADR-0168](0168-host-based-tenant-resolution.md), [ADR-0170](0170-tenant-scoped-pages.md), [ADR-0171](0171-tenant-scoped-news-content-and-jobs.md)

## Context

Site settings, navigation, redirects and form inbox records are shared by the
web shell and the client API. Leaving them global lets one tenant overwrite or
read another tenant's branding, routes or customer submissions.

## Decision

Migration `202609190008_site_tenant_scope` moves settings, navigation and
redirect identity to composite tenant keys and adds a tenant discriminator to
form inbox rows. The site SQL store always uses the empty tenant namespace for
single-customer mode and a validated context scope when shared tenancy is
enabled. Reads, writes, updates, idempotency lookups and bounded purges all
carry the same predicate.

## Consequences

- Client-facing settings/navigation and admin redirect/form workflows are
  isolated by tenant.
- Existing rows remain in the empty namespace and need controlled backfill
  before shared mode.
- The standalone forms module, auth/members and commerce stores still need
  their own tenant contracts and authorization certification.
