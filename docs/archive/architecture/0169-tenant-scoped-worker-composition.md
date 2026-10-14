# ADR-0169: Tenant-scoped worker composition

- Status: Accepted (static host-map scope source; dynamic provisioning open)
- Date: 2026-09-19
- Related decisions: [ADR-0162](0162-tenant-scoped-outbox-contract.md), [ADR-0163](0163-tenant-context-outbox-propagation.md), [ADR-0168](0168-host-based-tenant-resolution.md)

## Context

Storage primitives can fail closed when a tenant scope is missing, but that
property is lost if the composition root leaves `RequireTenantScope` disabled
or if a shared worker polls with `context.Background()`. In that case a
multi-tenant deployment can silently fall back to legacy global outbox and
audit operations.

## Decision

When `tenancy.enabled` is true, the composition root enables fail-closed tenant
scope requirements on audit, outbox and media stores. The outbox worker receives
the validated scopes derived from the operator host map and executes one bounded
claim/delivery pass per scope. Tenancy-disabled deployments retain the existing
single-customer worker behavior.

## Consequences

- Missing scope becomes an error instead of a cross-tenant fallback.
- A static host map is sufficient for the current single-process deployment.
- Tenant provisioning changes require rebuilding the worker scope list; dynamic
  provisioning and multi-instance scope distribution remain open.
- This does not replace PostgreSQL RLS or certify every repository call site.
