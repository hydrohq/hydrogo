# ADR-0347: Versioned public generated Admin runtime contract

- **Status:** Accepted
- **Date:** 2026-10-07
- **Supersedes:** None
- **Superseded by:** None
- **Related decisions:** [ADR-0061](0061-sdk-compatibility-assertions.md), [ADR-0210](0210-blueprint-scaffold-boundary.md), [ADR-0240](0240-generated-route-composition.md), [ADR-0346](0346-blueprint-admin-studio-workflows.md)

## Context

HydroGo documents Client Editions as independent Go modules, but the generated
Admin runtime was composed by importing `internal/adminstudio`. Go correctly
rejects that import outside the HydroGo module tree. A temporary independent
module using a local replacement reproduced the failure, so repository-local
Edition examples did not prove that a customer Edition could consume the
runtime.

Admin rendering needs stable typed metadata and a handler contract. Making
that contract public must not transfer authorization, tenant selection, CSRF,
route activation, migration approval, or data access to Blueprint metadata.

## Decision

- `extension/v1/adminstudio` is the public, versioned import path for the
  generated Admin runtime composition contract.
- The package exposes aliases for registry metadata, generated handler/options,
  workflow and editorial callback types, and the reviewed field/status
  constants. The aliases preserve existing typed Templ integration while
  keeping one runtime implementation inside Core.
- Client Editions must provide the API handler, authorization and
  resource-read policy, CSRF issuance/validation, page renderer, and any
  enabled editorial callbacks. Registration fails closed when required
  callbacks are absent.
- Blueprint generation remains proposal-only: it cannot infer policy, execute
  DDL, or register an inactive resource. Runtime route registration remains an
  explicit Edition composition call.
- Public contract changes require compile-time contract tests and a standalone
  external Go module build. Incompatible future changes use a new versioned
  import path; internal refactoring must preserve the current v1 signatures.

## Alternatives

- Make `internal/adminstudio` directly public by moving it: rejected because
  the existing implementation is shared with Core-only helpers and contains
  internal dependencies. Moving it would create unnecessary module churn.
- Generate the Admin handler into every Edition: rejected because it would
  duplicate security-sensitive runtime code and fragment fixes.
- Let Core discover and activate resources implicitly: rejected because
  activation and Edition-owned policy must remain explicit under ADR-0210 and
  ADR-0240.

## Consequences

### Benefits

- Independent Editions can compile against a supported import path.
- Runtime behavior stays shared, while the Edition explicitly binds its own
  authorization, tenant scope, CSRF, API, and shell.
- An external-module test catches regressions that monorepo examples hide.

### Costs and risks

- The exported v1 signatures become a compatibility promise and require
  compile-time fixtures.
- The facade exposes typed handler options that must be reviewed whenever
  their internal source types change.
- Editions still need a public or Edition-owned Admin shell; this contract
  intentionally supplies no customer-specific presentation or policy.

## Scale impact

- **Compact:** aliases and package wrappers add no services, goroutines,
  workers, routes, assets, or database resources until the Edition explicitly
  registers a handler.
- **Standard:** uses the Edition's existing API handler and connection pool;
  the facade introduces no additional pool or cache.
- **Distributed:** handlers remain stateless and use the Edition's resolved
  tenant context and service boundary. This decision does not enable shared
  multi-tenant SaaS.

## Database impact

- **SQLite:** no schema, query, transaction, migration, or adapter change.
- **PostgreSQL:** no schema, query, transaction, migration, or adapter change.
- **Future adapters:** no database behavior is introduced by this facade.

No migration is generated or required. Migration rollback metadata is not
applicable because persisted state does not change.

## Cache impact

No cache service or key is added. Existing private/no-store Admin response
behavior is unchanged; there is no TTL, invalidation, stampede, or fallback
change.

## Deployment and compatibility

The facade is compiled into the Edition binary and adds no runtime service or
asset. A Blue/Green rollout continues to serve the same handler and routes;
the only deploy-time requirement is that each Edition build resolves a
compatible HydroGo v1 module. Rollback is a normal binary rollback. There is no
database migration or shared mutable state to recover. CI compiles both the
HydroGo contract fixture and a nested standalone Edition module with a local
replacement, verifying that `internal` package restrictions are respected.

## Validation

- `go test ./extension/v1/adminstudio` compiles the public signatures and
  callback types.
- `go test -mod=mod ./...` from `tests/external-edition` compiles a separate Go
  module that parses a manifest and composes `GeneratedAdminHandler`.
- `go test ./...`, `moduleguard`, `migrationguard`, and `importguard` remain
  required repository gates. This change adds no imports between feature
  modules and no migration files.

## References

- [ADR-0061: Compile-time assertions for extension contracts](0061-sdk-compatibility-assertions.md)
- [ADR-0210: Blueprint scaffold boundary](0210-blueprint-scaffold-boundary.md)
- [ADR-0240: Generated route composition](0240-generated-route-composition.md)
- [ADR-0346: Blueprint Admin Studio and editorial workflows](0346-blueprint-admin-studio-workflows.md)
- `extension/v1/adminstudio`
- `tests/external-edition`
