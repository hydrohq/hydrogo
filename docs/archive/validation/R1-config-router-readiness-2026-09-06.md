# R1 config, router and readiness validation

**Date:** 2026-09-06  
**Roadmap item:** R1 runtime foundation  
**Status:** Partial success; bootstrap/session, CSRF and full audit security slice pending.

## Implemented behavior

- `hydro serve` and `hydro migrate` use the same typed runtime config.
- Config precedence for non-secret fields is CLI → environment → YAML → default. A YAML unknown field is a startup error.
- Database DSN secret can only be defined in one of YAML, environment, and CLI sources; duplicated secret sources produce a clear error.
- Unsupported dialect, invalid listen address, and missing DSN are rejected at startup.
- SQLite PRAGMAs are strictly applied on every connection by `database.Open` independent of the user's DSN format. Initial pool limits are 1/1 for SQLite and 10 open/5 idle connections for PostgreSQL; the actual profile defaults wait for benchmarks.
- HTTP startup does not run migrations. `hydro migrate` is a separate command.
- `/healthz` only reports process liveness. `/readyz` performs DB ping and migration ledger checksum checks within a two-second budget; returns 503 without leaking error details to the public response.
- The router enforces classifying every route as `Public` or `Protected`. Unclassified routes and protected routes without an authorizer are rejected during creation.
- HTTP read-header/read/write/idle timeouts, header byte limits, and graceful shutdown are configured.

## Local evidence

On Windows amd64 and Go 1.27.0, `go test ./...`, `go vet ./...`, CLI build, and SQLite `hydro migrate` smoke tests succeeded. The tests cover config precedence, unknown-key and secret conflict rejection, router default-deny behavior, readiness error hiding, and the DB contract.

## Boundaries

The current protected-route authorizer is only a test dependency proving the router contract; it is not an actual session. Bootstrap admin, Argon2id, DB-backed opaque session, CSRF, security headers, and append-only audit API will be implemented in the subsequent R1 security slice. Pool sizing per profile and config schema versioning also remain open.
