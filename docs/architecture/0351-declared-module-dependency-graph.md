# ADR-0351: Declare exact direct dependencies between HydroGo modules

- **Status:** Accepted
- **Date:** 2026-10-07
- **Supersedes:** None
- **Superseded by:** None
- **Related:** [ADR-0266](0266-design-system-blocks-and-interactions.md), [ADR-0341](0341-admin-family-overhaul.md), [ADR-0346](0346-blueprint-admin-studio-workflows.md), [ADR-0347](0347-versioned-public-generated-admin-runtime.md)

## Context

The module guard rejected eight existing direct imports from `members`, `news`,
`pages`, and `site` to the shared design system and site/SEO modules. ADR-0341
recorded the findings as pre-existing and unrelated to its UI change, but the
CI guard had no machine-readable record of the dependencies that the current
feature modules use. A broad allowlist for each affected module would hide all
future cross-module imports and weaken the guard.

The architecture already treats `modules/designsystem` as the typed shared
component library (ADR-0266). News and Pages also consume Site-owned settings
and SEO behavior. Those current direct relationships need an explicit contract
until a separately reviewed module extraction changes them.

## Decision

- `modules/dependencies.json` declares the exact direct module dependency graph.
- Every production Go import from one module to another must have a matching
  edge in that graph. Test-only imports do not affect the production graph.
- The guard rejects undeclared dependencies, references to missing modules,
  unused declarations, self-dependencies, and dependency cycles.
- Only the existing aggregate root `modules/api` remains explicitly
  allowlisted. Feature modules are never allowlisted as a whole.
- The initial graph records the eight existing imports: `members` and `site`
  depend on `designsystem`; `news` depends on `designsystem` and `site`; and
  `pages` depends on `designsystem`, `seo`, and `site`.
- Changing or removing an edge requires updating the graph and its tests in
  the same commit. A dependency that introduces a cycle or violates a higher
  level boundary requires a new architectural decision before implementation.

This records the current compile-time graph; it does not change runtime
registration, ownership, authorization, data access, or module activation.

## Alternatives

- Allowlist all four affected feature modules: rejected because it would permit
  arbitrary future dependencies from those modules.
- Remove all eight imports immediately: rejected because moving shared typed
  UI, Site-owned settings, and SEO behavior requires separate domain/API
  changes and is not necessary to restore precise CI enforcement.
- Ignore the guard failure in CI: rejected because moduleguard is a mandatory
  architectural safeguard.

## Consequences

### Benefits

- Existing compile-time dependencies are explicit and reviewable.
- New cross-module imports fail CI unless the graph is deliberately updated.
- Cycle detection prevents a declared dependency change from silently making
  feature modules mutually dependent.

### Costs and risks

- The initial graph preserves existing coupling; it does not claim those
  dependencies are ideal module boundaries.
- The graph must stay synchronized with production imports. The guard detects
  both missing and unused edges.

## Scale impact

- **Compact:** no runtime cost, services, workers, routes, assets, or memory
  overhead are added.
- **Standard:** no runtime or capacity change.
- **Distributed:** no runtime or deployment change.

## Database impact

- **SQLite:** no schema, query, migration, or adapter behavior changes.
- **PostgreSQL:** no schema, query, migration, or adapter behavior changes.
- **Future adapters:** no database behavior is introduced.

No migration or rollback metadata is required because no persisted state
changes.

## Cache impact

No cache keys, TTLs, invalidation, stampede behavior, or fallback paths change.

## Deployment and compatibility

The JSON graph is read by the build-time CI guard and is not embedded in
production binaries. Deployments, Blue/Green compatibility, sessions, media,
assets, and jobs are unchanged. Rollback restores only the previous guard and
graph sources; no data recovery is required.

## Validation

- Unit tests prove declared edges pass, undeclared imports fail, and dependency
  cycles fail.
- The production command `go run ./tools/moduleguard -root modules -allow
  modules/api -dependencies modules/dependencies.json` passes against the
  current graph and remains a mandatory CI step.
- `go test ./tools/moduleguard` and `go test ./...` validate the implementation.
- Migrationguard, database contract tests, capacity profiles, and Edition
  importguards are unchanged; they remain mandatory independent gates.

## References

- [ADR-0266: Design-system blocks and lightweight interactions](0266-design-system-blocks-and-interactions.md)
- [ADR-0341: Admin family overhaul](0341-admin-family-overhaul.md)
- [ADR-0346: Blueprint Admin Studio and editorial workflows](0346-blueprint-admin-studio-workflows.md)
- [ADR-0347: Versioned public generated Admin runtime contract](0347-versioned-public-generated-admin-runtime.md)
- `modules/dependencies.json`
- `tools/moduleguard`
