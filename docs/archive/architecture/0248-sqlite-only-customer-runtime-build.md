# ADR-0248: SQLite-only customer runtime build variant

- **Status:** Accepted
- **Date:** 2026-09-22
- **Related:** ADR-0006, ADR-0007, ADR-0010, ADR-0143, ADR-0144, ADR-0214, ADR-0247

## Context

ADR-0247 introduced import and artifact budgets for thin generated Edition
fixtures. The customer runtime builder still always links both SQLite and
PostgreSQL drivers, even when an Edition deployment is intentionally a
single-node SQLite installation. This adds avoidable binary and dependency
surface, while silently omitting PostgreSQL in a binary that claims to support
it would be unsafe.

## Decision

Customer package scripts accept `HYDRO_DATABASE_VARIANT=dual|sqlite`. The
default remains `dual` for compatibility. `sqlite` compiles with the
`hydro_compact` build tag and includes SQLite but not pgx. Any configuration
that selects PostgreSQL in that artifact must fail during configuration
validation with an explicit unavailable-driver error. `hydro` database open
also retains the same fail-closed check.

The build variant is recorded in `PACKAGE-MANIFEST.md`. This tag controls only
database driver registration; it does not select modules, change schema,
perform DDL, or introduce runtime plugins. PostgreSQL-capable Editions must be
packaged as `dual`.

## Alternatives

- Always link both drivers: keeps maximum flexibility but retains avoidable
  dependency and binary cost in SQLite-only deployments.
- Runtime plugin loading: rejected because it weakens the self-contained
  artifact and complicates supply-chain verification.
- Infer variant from configuration after build: impossible to remove a driver
  from an already-linked binary and easy to misrepresent package capability.

## Consequences

### Benefits

- SQLite-only customer runtimes exclude pgx and its transitive driver packages.
- Unsupported PostgreSQL configuration fails before serving traffic.
- Default dual-driver builds remain source and behavior compatible.

### Costs and risks

- A package's database capability is fixed at build time and must be tracked
  through release and deployment records.
- Switching such an Edition to PostgreSQL requires a dual-driver artifact and
  normal migration/recovery planning.
- This does not reduce unrelated Workspace dependencies or certify a complete
  customer Edition's footprint.

## Scale impact

- Compact: no extra services, pools, or workers; one fewer SQL driver.
- Standard: variant is an artifact choice, not a profile choice.
- Distributed: PostgreSQL is required; use the dual-driver build.

## Database impact

- SQLite: remains supported in both variants.
- PostgreSQL: supported only in `dual`; compact variant rejects it.
- Future adapters: each requires its own explicit build capability and
  certification; no implicit adapter loading is introduced.

## Cache impact

None; cache contracts, keys, TTLs, invalidation, stampede protection, and
fallback behavior are unchanged.

## Deployment and compatibility

No migration is introduced. Blue/green deployments must use an artifact whose
driver set covers the configured dialect. Existing packages retain `dual` by
default. Rollback is to the previous binary/package; schema rollback remains
subject to the migration's recovery notes.

## Validation

CI verifies SQLite-only driver absence, explicit PostgreSQL rejection, and
SQLite behavior. Thin Edition footprint checks cover both dual and SQLite-only
variants. Customer package build scripts record the selected variant.

## References

- [Generated Edition footprint regression budget](0247-generated-edition-footprint-budget.md)
- [Core footprint and Edition variant measurement](../validation/R123-sqlite-only-build-2026-09-22.md)
