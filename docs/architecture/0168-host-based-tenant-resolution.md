# ADR-0168: Host-based tenant resolution

- Status: Accepted (single-process resolver; provisioning and public-domain operations open)
- Date: 2026-09-19
- Related decisions: [ADR-0154](0154-tenant-scope-primitive.md), [ADR-0163](0163-tenant-context-outbox-propagation.md), [ADR-0165](0165-tenant-scoped-media-metadata.md)

## Context

Tenant-aware repositories require a trusted scope at the HTTP boundary. Taking
tenant IDs from headers, query parameters, or paths lets a caller select another
tenant and turns every downstream filter into an IDOR risk.

## Decision

An operator-configured `tenancy.hosts` map resolves normalized request hosts to
validated tenant IDs. When tenancy is enabled, unknown hosts fail closed with
HTTP 421; health/readiness endpoints remain tenant-neutral for load balancers.
The resolver ignores tenant headers and attaches scope before route handlers,
repositories, media and background handoff code run. Single-customer deployments
leave tenancy disabled and incur no resolver work.

## Consequences

- Tenant context is established at the earliest trusted application boundary.
- DNS/TLS, host allowlist, tenant provisioning, custom domains and proxy host
  preservation require deployment controls and drills.
- Background workers still need an explicit tenant source; they cannot infer
  scope from an arbitrary request header.
