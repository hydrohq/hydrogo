# ADR-0348: Single-call explicit generated Admin activation

- **Status:** Superseded by [ADR-0349](0349-explicit-generated-resource-activation.md)
- **Date:** 2026-10-07
- **Supersedes:** None
- **Superseded by:** None
- **Related decisions:** [ADR-0210](0210-blueprint-scaffold-boundary.md), [ADR-0240](0240-generated-route-composition.md), [ADR-0346](0346-blueprint-admin-studio-workflows.md), [ADR-0347](0347-versioned-public-generated-admin-runtime.md)

## Context

ADR-0347 makes the generated Admin runtime available to Client Editions through
a versioned public package. Edition code still repeats the same safe
mechanical steps: parse its embedded generated manifest, construct the shared
handler, and register its routes. Repetition slows the first working Edition
without adding a policy decision.

## Decision

- The public `extension/v1/adminstudio` package provides
  `RegisterGenerated(mux, manifest, options)` to parse the provided manifest,
  build the shared handler, and register its existing Admin routes.
- The Edition must call this function explicitly. The generator and package
  never register routes during package initialization or process startup by
  discovery.
- The Edition supplies all security and integration callbacks through
  `GeneratedAdminOptions`. Missing required callbacks cause registration to
  fail closed. The helper does not invent authorization, tenant scope, CSRF,
  audit, workflow, migration, or shell behavior.
- A malformed manifest returns its validation error and registers no routes.
  Registration errors are returned to the Edition caller for startup handling.

## Alternatives

- Generate a separate handler implementation for each Edition: rejected
  because it duplicates shared runtime code.
- Register from package initialization or global state: rejected because an
  inactive resource must not add routes or work and activation must stay
  reviewable in Edition composition.
- Keep manifest parsing and handler registration duplicated in each Edition:
  rejected because those steps have no Edition-specific policy content.

## Consequences

### Benefits

- The repetitive activation path is one typed call while policy remains
  Edition-owned.
- A standalone external Edition test can exercise the actual registered
  handler route using only public versioned packages.

### Costs and risks

- The helper is part of the v1 compatibility promise.
- Edition startup must still provide and validate its own policy callbacks.
- The helper registers existing routes, so Editions remain responsible for
  composing it into the intended mux and route group.

## Scale impact

- **Compact:** no background service, worker, asset, or runtime cost exists
  until the Edition explicitly calls the helper. Registration adds only the
  selected handler routes.
- **Standard:** reuses the supplied API handler and pool; no added database
  connections or cache.
- **Distributed:** stateless route composition continues to use the Edition's
  request and tenant context.

## Database impact

- **SQLite:** no schema or query changes; no migration is required.
- **PostgreSQL:** no schema or query changes; no migration is required.
- **Future adapters:** no database behavior is added.

There is no rollback metadata because this decision persists no state.

## Cache impact

No cache key, TTL, invalidation, stampede, fallback, or cache service changes.
Admin responses retain their current private/no-store behavior.

## Deployment and compatibility

The helper is a compile-time API over the same handler binary and adds no
runtime service. Blue/Green and rollback behavior remain ordinary binary
deployment; no database or asset migration is involved. Editions using the
helper require a HydroGo v1 module that exports this function. The external
Edition module test exercises manifest parsing, registration, and an HTTP GET
through the registered generated Admin route.

## Validation

- `go test ./extension/v1/adminstudio` compiles the public API contract.
- `go test -mod=mod ./...` in `tests/external-edition` executes a generated Admin
  HTTP list route from a separate Go module.
- Repository Go, import, module, and migration guards remain mandatory.

## References

- [ADR-0347: Versioned public generated Admin runtime contract](0347-versioned-public-generated-admin-runtime.md)
- `extension/v1/adminstudio`
- `tests/external-edition`
