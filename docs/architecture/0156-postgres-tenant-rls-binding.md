# ADR-0156: Transaction-local PostgreSQL tenant binding

- Status: Accepted (foundation only; RLS policies and shared mode remain open)
- Date: 2026-09-19
- Related decisions: [ADR-0140](0140-tenant-isolation-scope.md), [ADR-0149](0149-shared-multi-tenant-saas-evolution.md), [ADR-0154](0154-tenant-scope-primitive.md)

## Decision

`internal/tenant.BindPostgresTransaction` is the narrow SQL boundary for
binding an already validated tenant scope to a PostgreSQL transaction. It uses
parameterized `set_config('app.current_tenant_id', $1, true)`, making the value
transaction-local and automatically cleared at commit/rollback. Missing scope
or executor fails closed.

Domain and application services do not construct SQL or interpolate tenant IDs;
the adapter owns this dialect detail. Callers must bind before any
tenant-scoped repository query. Actual table policies, migration metadata,
RLS role privileges, and call-site propagation remain separate gates.

## Consequences

- PostgreSQL RLS policies have a consistent, injection-safe session variable
  boundary.
- Scope cannot leak across pooled connections after transaction completion.
- The primitive alone does not enable shared tenancy or prove every query is
  protected; unbound repository paths must still be inventoried and rejected.
