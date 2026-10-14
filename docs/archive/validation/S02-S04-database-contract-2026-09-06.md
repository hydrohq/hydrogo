# S02/S04 database transaction and migration validation

**Date:** 2026-09-06  
**Roadmap item:** R1 / S02–S04 DB section  
**Status:** Succeeded; backup/encoder and full support matrix await separate R1/P1 gates.

## Selections

- SQLite: `modernc.org/sqlite` v1.58.0. Pure Go driver covering Windows amd64 and Linux amd64 targets; does not require cgo toolchain. The linked `modernc.org/libc` version of the driver is pinned at v1.75.6 in the module graph.
- PostgreSQL: `github.com/jackc/pgx/v5` v5.10.0, via the `database/sql` compatibility layer.
- Migration: No external migration framework was added in the initial experiment. Dialect SQL, immutable SHA-256 metadata, and ledger behavior within a transaction were proven with a small HydroGo runner. The necessity of a broader tool will be evaluated with subsequent migration examples.

Sources: [modernc SQLite package docs](https://pkg.go.dev/modernc.org/sqlite), [pgx stdlib package docs](https://pkg.go.dev/github.com/jackc/pgx/v5/stdlib).

## Environment and result

| Target | Database | Result |
|---|---|---|
| Windows amd64, Go 1.27.0 | File-based SQLite / WAL | Succeeded |
| GitHub Actions Ubuntu, Go 1.27.0 | PostgreSQL 18 isolated service container | Succeeded |

CI evidence: [run 33993049815](https://github.com/ardaozyildirim/hydrogo/actions/runs/33993049815).

## Proven contract

- SQLite applied `foreign_keys=on`, `journal_mode=WAL`, `synchronous=NORMAL`, and `busy_timeout=5000` values on every connection. The `file:///C:/...` URI fix for the Windows drive letter was found and persisted during testing.
- Pages draft, audit event, and outbox event used the same `sql.Tx` executor. After an error injected into the outbox, no rows remained in any of the three tables.
- In a successful call, the three records were committed together.
- Of two concurrent updates on the same page/version, only one changed a row; a silent last-writer-wins did not occur.
- A second run of the same migration was a no-op. When the SQL content was altered, the ledger checksum mismatch rejected the operation.
- Two dialect checksums in the migration metadata were tied to actual SQL files via CI testing.

## Limits

This experiment is not a production schema. The PostgreSQL test wrote to the `hydro_contract` database created in each CI run, not a customer DB. The local PostgreSQL service was not used since no password was provided. Deadlock/serialization retry, online DDL, backup/restore, connection pool capacity, and the supported PostgreSQL minor version matrix are not yet certified.
