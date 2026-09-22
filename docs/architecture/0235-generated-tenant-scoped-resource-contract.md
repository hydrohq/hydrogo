# ADR-0235: Generated tenant-scoped resource contract

- **Status:** Accepted
- **Date:** 2026-09-22
- **Related decisions:** [ADR-0233](0233-public-tenant-scope-contract.md), [ADR-0229](0229-generated-bounded-list-transport.md)

## Decision

Blueprints may opt a resource into `tenant_scoped: true`. The generator then
emits a tenant-aware model, repository, service and HTTP transport under the
machine-owned `generated/` boundary. Every read and write receives the tenant
identifier explicitly; the transport fails closed unless the Edition injects
an authenticated tenant resolver and policy.

The generated repository applies an explicit tenant predicate for SQLite and
PostgreSQL and preserves the validated bounded query AST for filters, sorting
and pagination. PostgreSQL RLS remains an adapter/deployment control, not a
replacement for the generated predicate. Tenant provisioning, membership,
cross-tenant administration and policy decisions remain Edition-owned.

The generator only proposes the migration column (`tenant_id TEXT NOT NULL`)
and never executes DDL. Editions must add indexes, RLS policies, rollback
metadata and certified SQLite/PostgreSQL tests before enabling the resource.

## Consequences

- A resource cannot accidentally fall back to an unscoped repository method.
- The generated API is explicit and testable, but existing non-tenant
  resources retain their compatible signatures.
- Tenant-aware code generation does not claim complete isolation: database
  roles, provisioning idempotency, worker propagation, quotas and restore
  drills remain required production gates.

## Verification

`TestGeneratedTenantScopedResourceBindsTenantContract` checks the generated
model, repository, service, transport and migration markers. Full repository
tests and `go vet ./...` must pass before an Edition consumes the contract.
