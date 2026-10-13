# R110 — Core Over-Engineering Audit

**Date:** 2026-09-22  
**Scope:** Whole repository, complexity/bloat only  
**Status:** Lean boundary review; no safe deletion applied

The Ponytail whole-repo scan evaluated the following candidates:

- `yagni` `extension.Module` legacy contract overlaps with the new `extension/v1.Module`;
  however, it cannot be deleted today due to an explicit deprecation/compatibility decision.
  Replacement: a single v1 contract once the migration window concludes.
- `yagni` `observability.Exporter` does not carry a single built-in provider; this is a
  provider-neutral boundary, not unnecessary abstraction. Replacement:
  none; preserved for external OTLP/exporter adapters.
- `yagni` `ui.DesignSystem` registry contains several class mappings; this is a
  deliberate boundary separating Client Edition provider switching from Core.
  Replacement: none.
- `yagni` `database.Executor`/`QueryExecutor` appears minimal; it must not be deleted as it is
  required for SQLite/Postgres adapter isolation and test doubles.
  Replacement: none.
- `stdlib`/`native`: no production dependency in the dependency manifest was identified that could be safely
  replaced directly with stdlib; templ, pgx,
  Redis, S3, OTEL, SQLite, and YAML represent authentic capability boundaries.
- `x/net/idna`, `x/text/currency`, and OTEL exporter packages checked directly via `go mod why`
  resolve to real production imports; these are not transitive/dead dependencies.

**Result:** Lines that can be safely deleted without breaking behavior: 0; dependencies that can be
safely removed: 0. A dedicated removal ADR may be opened in the future for deprecated `extension.Module`;
removing it today would break Edition compatibility.
