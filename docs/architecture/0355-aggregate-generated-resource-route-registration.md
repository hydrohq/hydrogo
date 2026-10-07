# ADR-0355: Aggregate generated resource route registration

- **Status:** Accepted
- **Date:** 2026-10-07
- **Supersedes:** None
- **Superseded by:** None
- **Related decisions:** [ADR-0210](0210-blueprint-scaffold-boundary.md), [ADR-0240](0240-generated-route-composition.md), [ADR-0346](0346-blueprint-admin-studio-workflows.md), [ADR-0349](0349-explicit-generated-resource-activation.md), [ADR-0354](0354-blueprint-derived-admin-activation.md)

## Context

Blueprint generation already aggregates Admin metadata and provides one
explicit Admin activation helper. API activation still required the Edition to
import and register every generated resource separately. A newly generated
resource could therefore appear in the Admin registry while its API route was
missing from an Edition whose APIs shared the same persistence and policy
composition.

The generator can remove this repeated wiring only when the Edition supplies
the actual dependencies. It must not infer authorization, tenant ownership,
middleware, migration approval, or module lifecycle.

## Decision

- When the generated output root is inside a Go module, the generator emits a
  deterministic `generated.RegisterGeneratedResources` helper that covers
  every resource in the aggregate generated registry.
- Calling this helper is an explicit Edition activation decision. It activates
  the complete generated resource set; an Edition that enables only a subset
  continues to call each selected resource's `Register<Name>Route` directly.
- The helper requires the Edition database, dialect, and a `PolicyFor` callback
  that returns each resource's generated policy contract. A missing policy
  fails before any route is registered.
- If any generated resource is tenant-scoped, the helper requires an
  Edition-owned `ResolveTenant` callback. Tenant resolution remains
  request-scoped; an absent resolver fails before any route is registered.
- Resource policies and tenant resolvers are gathered for the complete
  generated set before route registration starts. This prevents a missing
  dependency from leaving a partially activated route set.
- The helper never grants permissions, installs authentication/CSRF/rate-limit
  middleware, applies DDL, starts workers, or resolves workflow behavior.
  Existing individual route registration remains the lower-level contract.
- The generated helper is checked by `hydro project check` against the current
  resource specs and Go module path. A stale or hand-edited helper fails closed.

## Alternatives

- Keep one import and route-registration call per resource: rejected because
  resource generation can silently omit a resource from an already composed
  Edition API.
- Register every route from package initialization: rejected because route
  lifecycle must remain an explicit Edition decision and partial activation
  must remain possible.
- Derive default authorization from Blueprint permissions or Admin metadata:
  rejected because presentation/schema metadata cannot grant access.
- Apply generated migrations while registering routes: rejected because
  migration proposals require operator review and independent recovery checks.

## Consequences

### Benefits

- A fully enabled Edition adds a new Blueprint resource to its API route set
  after one regeneration, with no new per-resource Go wiring.
- Policy and tenant dependencies are checked before any route is added.
- Existing route paths, service/repository behavior, and direct per-resource
  activation remain unchanged.

### Costs and risks

- The aggregate helper imports every generated resource package; Editions
  requiring a partial build/runtime footprint should use direct registration.
- The shared policy callback must dispatch each resource ID correctly. A
  permissive Edition callback remains the Edition's responsibility and risk.
- Standalone generated output without a Go module cannot resolve child package
  import paths, so it continues to use direct route registration until it is
  placed in a module.

## Scale impact

- **Compact:** the aggregate helper is a generated source file and adds no
  runtime cost unless the Edition imports and calls it. Calling it activates
  every generated API route; partial Editions retain per-resource calls.
- **Standard:** routes reuse the Edition's existing `ServeMux`, SQL pool, and
  middleware stack. No service or worker is created per request.
- **Distributed:** route selection is fixed at startup. Policy and tenant
  callbacks continue to evaluate request context through Edition-owned
  services.

## Database impact

- **SQLite:** no query, schema, transaction, or migration behavior changes.
  Existing generated SQLite API/Admin contract tests exercise the aggregate
  helper.
- **PostgreSQL:** no query or schema behavior changes. The same generated
  Edition contract test exercises the helper against an isolated schema when
  `HYDRO_TEST_POSTGRES_URL` is supplied.
- **Future adapters:** the helper only forwards the Edition database handle
  and dialect into existing per-resource route registration; adapter
  certification remains unchanged.

No migration is added. Existing generated migration compatibility and
recovery metadata remain governed by each resource's migration contract.

## Cache impact

No cache keys, TTLs, invalidation, stampede behavior, fallback, or cache service
changes.

## Deployment and compatibility

The generated helper is additive source code. `hydro project check` validates
that its imports and route list match the aggregate resource specs. Calling it
changes the Edition route set only after rebuild/redeployment. The aggregate
and direct APIs expose the same existing route patterns and transport
contracts; there is no data migration. Rolling back the binary restores the
previous API route set. Generated directories without a Go module do not
receive the helper and retain their current source contract.

## Validation

- Generator tests verify deterministic multi-resource imports, policy
  preflight, tenant-resolver requirements, and rejection of stale generated
  helper source.
- Project Edition SQLite tests create/read resources through aggregate routes
  and the generated Admin; a missing policy for one resource proves no route
  from the set was registered.
- Project Edition PostgreSQL tests run the same aggregate API/Admin flow in an
  isolated schema when the PostgreSQL test DSN is configured.
- `hydro project check`, `go test ./internal/generator`, `go test
  ./examples/project-edition/...`, moduleguard, migrationguard, and Edition
  importguards remain required.

## References

- [Blueprint Admin activation](0354-blueprint-derived-admin-activation.md)
- [Blueprint first CRUD measurement](../validation/blueprint-first-crud-measurement-2026-10-07.md)
- `internal/generator/route_registration.go`
- `examples/project-edition/generated/route_registration.go`
