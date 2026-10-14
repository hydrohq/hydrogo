# ADR-0167: Tenant-scoped media derivative jobs

- Status: Accepted (queued-job bridge; object provider propagation open)
- Date: 2026-09-19
- Related decisions: [ADR-0166](0166-tenant-object-storage-namespace.md), [ADR-0162](0162-tenant-scoped-outbox-contract.md)

## Context

Media optimization runs asynchronously after an upload. A background job that
loses tenant context can write derivative metadata into the wrong partition even
when the original upload was correctly namespaced.

## Decision

The optimizer restores a validated tenant scope from the immutable
`tenants/{tenant_id}/...` storage key before derivative repository calls. The
media SQL store scopes variant state transitions and namespaces ready derivative
keys through the same helper. Legacy unprefixed assets retain the existing
single-customer job behavior.

## Consequences

- Queued derivative metadata follows the tenant boundary without adding secrets
  or mutable tenant state to the job payload.
- Legacy assets require an explicit backfill before shared mode can be enabled.
- Provider-backed derivative writes, queue/outbox tenant propagation and
  cross-tenant job/property tests remain open.
