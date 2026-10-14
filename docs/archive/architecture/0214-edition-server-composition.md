# ADR-0214: Client Edition Server Composition and Reviewed Migration Boundary

- **Status:** Accepted
- **Date:** 2026-09-21
- **Decision:** Client Editions may ship a thin, explicit server composition
  command that imports generated contracts and custom policy. The command has
  separate `migrate` and `serve` subcommands. `migrate` executes the reviewed,
  embedded `generated.MigrationSQL` only when the operator invokes it; `serve`
  never performs DDL and only opens the already-prepared database.

## Context

The Workspace generator produces persistence and transport contracts, but a
generated collection cannot safely be assumed to be the full HydroGo Core
application. Editions need a reproducible way to prove that generated routes
can be composed into a single binary while keeping authorization and database
lifecycle decisions outside `generated/`.

## Consequences

- Project and Article examples now build as standalone Go binaries and expose
  only their generated route plus health check.
- The migration is available in the binary for operator-controlled execution,
  but startup does not mutate schema; this preserves the no-runtime-DDL rule.
- The examples use an explicitly named demo policy and are verification
  harnesses, not production tenancy/RBAC implementations. A production Edition
  must replace that policy and bind tenant scope before exposing the route.
- `scripts/verify-edition-servers.ps1` proves migrate → serve → HTTP POST/GET
  for both editions on SQLite. The shared PostgreSQL contract suite remains the
  database certification gate.
