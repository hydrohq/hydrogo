# ADR-0163: Tenant-context outbox propagation

- Status: Accepted (context bridge; shared-mode call-site audit open)
- Date: 2026-09-19
- Related decisions: [ADR-0162](0162-tenant-scoped-outbox-contract.md), [ADR-0154](0154-tenant-scope-primitive.md)

## Context

Existing module services already pass `context.Context` through transaction and
outbox interfaces. Requiring every current single-customer call site to switch
interfaces at once would create unnecessary compatibility risk, while allowing a
shared worker to silently use the legacy outbox path would be unsafe.

## Decision

The SQL outbox store automatically selects scoped operations whenever a valid
tenant scope is present in context. A `RequireTenantScope` store option provides
an explicit fail-closed mode for shared deployments: missing scope returns
`tenant.ErrMissingScope` instead of writing or claiming the legacy empty tenant
partition. Existing contexts without a scope retain the single-customer legacy
behavior until the deployment enables shared mode.

## Consequences

- Tenant context propagates through existing service interfaces without changing
  every module signature.
- Shared-mode composition roots must set `RequireTenantScope` and establish
  scope before invoking repositories, workers or webhooks.
- A complete call-site audit, tenant resolver at the router, PostgreSQL RLS and
  background worker composition remain required before shared tenancy launch.
