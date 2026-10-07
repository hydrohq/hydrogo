# ADR-0354: Blueprint-derived generated Admin activation helper

- **Status:** Accepted
- **Date:** 2026-10-07
- **Supersedes:** None
- **Superseded by:** None
- **Related decisions:** [ADR-0210](0210-blueprint-scaffold-boundary.md), [ADR-0240](0240-generated-route-composition.md), [ADR-0347](0347-versioned-public-generated-admin-runtime.md), [ADR-0349](0349-explicit-generated-resource-activation.md), [ADR-0353](0353-public-generated-admin-studio-shell.md)

## Context

Blueprint generation aggregates collection metadata in
`generated/admin_registry.json`, but Edition composition had to repeat each
resource ID in a separate `RegisterGenerated` call. Adding a Blueprint resource
could therefore produce its Admin views without adding it to the active Admin
list, leaving the user to discover and edit a second hand-maintained list.

## Decision

- The generator emits `generated/admin_registration.go` with
  `RegisterGeneratedAdmin(mux, options)`. The helper's explicit invocation is
  the Edition's Admin activation point; its selected IDs are generated from the
  Edition's current aggregate Admin registry in deterministic order.
- Adding a generated resource updates both the aggregate registry and this
  registration helper. `hydro project check` validates that the helper matches
  the generated resource specs and rejects missing, tampered, or stale output.
- Edition-owned `API`, `Authorize`, `CanReadResource`, CSRF callbacks,
  `RenderPage`, tenant resolution, migrations, and workflow services remain
  explicit inputs and are never inferred by the generator.
- Editions that intentionally activate only a subset continue to call the
  lower-level versioned `adminstudio.RegisterGenerated` with a reviewed list.
- A resource still has no route unless the Edition explicitly calls the
  generated registration helper from its composition root. Generation does not
  execute DDL, install middleware, or start a worker or service.

## Alternatives

- Keep a hand-maintained list in every Edition: rejected because Blueprint
  resources could be generated but omitted from Admin registration.
- Register every resource from package initialization: rejected because route
  activation and runtime footprint must stay under Edition startup control.
- Infer authorization from Admin metadata or presentation identity: rejected
  because UI structure is not a security policy.

## Consequences

### Benefits

- A newly defined Blueprint resource enters the generated Admin registry and
  registration list together; an already wired Edition exposes its collection
  in the shared panel without another per-resource ID edit.
- The Edition composition root retains one explicit activation call and owns
  all security and persistence dependencies.
- Compatibility validation detects stale generated activation output.

### Costs and risks

- The generated helper selects every resource in the aggregate generated Admin
  registry. Editions needing a partial deployment must use the lower-level
  explicit-selection API.
- Generated files depend on the versioned HydroGo Admin Studio v1 package.

## Scale impact

- **Compact:** registers only generated resource handlers during explicit
  Edition startup; no additional service, worker, pool, or browser runtime.
- **Standard:** reuses the Edition's existing mux, API, and connection pool.
- **Distributed:** active resource selection is immutable startup composition;
  request authorization and tenant scope remain request-specific Edition code.

## Database impact

- **SQLite:** no schema/query behavior changes.
- **PostgreSQL:** no schema/query behavior changes.
- **Future adapters:** no database behavior is introduced.

No migration or rollback DDL is required.

## Cache impact

No cache keys, TTL, invalidation, stampede, or fallback behavior changes.

## Deployment and compatibility

The generated helper is an additive source artifact. Editions regenerate and
rebuild against the existing versioned v1 runtime. Old binaries and persisted
data remain compatible. No routes change until an Edition rebuilds and invokes
the helper; binary rollback restores the previous route set.

## Validation

- Generator tests prove resource IDs are sorted and automatically track two
  Blueprint resources, and compatibility checks reject a stale helper.
- The Project Edition compiles and runs through the generated helper.
- The external Edition test and generated Admin Chromium CRUD flow verify
  shared shell/navigation and the resulting route behavior.
- `go test ./...`, `go vet ./...`, moduleguard, migrationguard, and Edition
  importguards remain mandatory.

## References

- [Admin Studio product audit and plan](../planning/ADMIN_STUDIO_PRODUCT_AUDIT_AND_PLAN.md)
- [Blueprint-to-Admin validation](../validation/blueprint-derived-admin-activation-2026-10-07.md)
- `internal/generator`
- `extension/v1/adminstudio`
