# ADR-0039: Conditional cross-module migration scopes

- **Status:** Accepted
- **Date:** 2026-09-14
- **Related decisions:** [ADR-0011](0011-consistency-upgrade-and-recovery.md), [ADR-0033](0033-runtime-module-resource-gating.md)

## Context

Some maintenance migrations protect several optional module tables at once. A
filtered Compact profile must not run such a migration when one of its target
tables is absent: SQLite cannot create a trigger on a missing table, and
creating inactive-module tables would violate the zero-footprint contract.

## Decision

Migration metadata may declare `requires_modules`. When the runtime applies or
checks a selected module set, a migration is selected only when its owning
module is enabled (core migrations remain eligible) and every declared module
is enabled. Unfiltered migration tooling retains the legacy behavior and
applies all migrations. The migration ledger still records only migrations
actually applied; skipped optional scopes remain compatible with a later
activation of the required modules.

## Consequences

- Minimal profiles do not create tables, triggers, or indexes for inactive
  modules merely because a shared maintenance migration exists.
- A cross-module protection migration is intentionally deferred until all of
  its target modules are active; module-specific protection should use a
  separately scoped migration when partial activation needs stronger fencing.
- SQLite and PostgreSQL selection/check behavior share the same metadata
  contract; each dialect still requires its own migration SQL and certification.

## Verification

`internal/database/migrate_test.go` applies the real embedded migrations to an
isolated SQLite database with only Pages and Media enabled, asserts that
Commerce and News tables are absent, and runs the filtered compatibility check.
The same suite verifies a core-only selection does not execute the
Pages-dependent audit-column migration.
PostgreSQL and production profile certification remain open gates.
