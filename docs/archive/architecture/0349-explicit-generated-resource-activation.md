# ADR-0349: Explicit generated resource activation

- **Status:** Accepted
- **Date:** 2026-10-07
- **Supersedes:** [ADR-0348](0348-generated-admin-explicit-activation-helper.md)
- **Superseded by:** None
- **Related decisions:** [ADR-0210](0210-blueprint-scaffold-boundary.md), [ADR-0240](0240-generated-route-composition.md), [ADR-0346](0346-blueprint-admin-studio-workflows.md), [ADR-0347](0347-versioned-public-generated-admin-runtime.md)

## Context

The single-call `RegisterGenerated` helper reduced Edition composition, but
accepting a manifest without an explicit resource selection could register
routes for generated resources that an Edition did not intend to activate.
Route absence is a required protection for inactive modules and resources.

## Decision

- `RegisterGenerated` requires a non-empty explicit list of active resource
  IDs in addition to the generated manifest and Edition-owned handler options.
- The helper rejects duplicate, empty, or unknown IDs and activates only the
  selected collections. A selected registry is validated before any route is
  registered.
- A resource not selected must have no generated Admin route in the mux.
- API, authorization, read policy, CSRF, shell rendering, tenant scope, and
  editorial services remain Edition-owned. The active list is selection
  metadata and grants no policy permission.
- The earlier versioned helper in ADR-0348 is superseded because its signature
  did not express the zero-route requirement for unselected resources.

## Alternatives

- Treat every resource in the manifest as selected: rejected because aggregate
  Edition manifests can include resources that are disabled for a deployment.
- Infer activation from route authorization or hidden navigation: rejected
  because an inaccessible route is still a route and presentation metadata is
  not an activation boundary.
- Discover activation from package initialization: rejected because it makes
  route lifecycle implicit and adds cost for inactive resources.

## Consequences

### Benefits

- Unselected generated resources do not receive Admin handlers or URLs.
- Edition code retains one explicit call and a short reviewable activation
  list.
- A standalone module test can assert the inactive route returns not found.

### Costs and risks

- Editions must maintain an active-resource list alongside generated metadata.
- The public v1 helper signature is a compatibility promise; activation rules
  need contract tests.

## Scale impact

- **Compact:** only selected resources are registered. No worker, service,
  asset, connection pool, or database resource is created by activation.
- **Standard:** selected handlers reuse Edition-owned APIs and pools.
- **Distributed:** selection is immutable startup configuration; request-time
  tenant scope still comes from the Edition's resolver.

## Database impact

- **SQLite:** no schema, query, or transaction changes; no migration required.
- **PostgreSQL:** no schema, query, or transaction changes; no migration
  required.
- **Future adapters:** no database behavior is added.

No persisted state changes, so migration rollback metadata is not applicable.

## Cache impact

No cache keys, TTL, invalidation, stampede, fallback, or cache services change.

## Deployment and compatibility

The selected routes are registered at Edition startup only after explicit
configuration. Blue/Green and binary rollback are unchanged; there are no
database or asset migrations. Editions must rebuild against the updated v1
contract. The external-module contract test verifies the selected route works
and a valid but unselected resource returns 404.

## Validation

- `go test ./extension/v1/adminstudio` verifies the public signature.
- `go test -mod=mod ./...` in `tests/external-edition` verifies successful
  selected registration and absence of an unselected route.
- `go test ./...`, `moduleguard`, `migrationguard`, and `importguard` remain
  required repository checks.

## References

- [ADR-0348: Single-call explicit generated Admin activation](0348-generated-admin-explicit-activation-helper.md)
- `extension/v1/adminstudio`
- `tests/external-edition`
