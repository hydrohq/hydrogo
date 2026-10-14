# ADR-0282: PostgreSQL physical schema diff

- **Status:** Accepted
- **Date:** 2026-09-28
- **Related:** [ADR-0006](0006-extensible-database-adapters.md),
  [ADR-0275](0275-live-migration-ledger-diff.md),
  [ADR-0277](0277-sqlite-physical-schema-diff.md)

## Context

The SQLite physical diff does not identify PostgreSQL catalog drift. Operators
need the same migration-selected expected schema compared to the configured
PostgreSQL schema, including columns, indexes, and constraints.

## Decision

`hydro db diff` may build the expected PostgreSQL catalog in a randomly named
scratch schema on the configured database connection. It applies only the
selected embedded migrations, compares normalized table, view, column, index,
and primary/unique/check/foreign-key constraint metadata, restores the
connection's original `search_path`, and drops the scratch schema before
returning. It never copies application rows or changes objects in the inspected
schema. A cleanup failure is returned as an error.

The configured database role must be able to create and drop a schema. The
operation uses one existing pooled connection and is bounded by the embedded
migration bundle and catalog size; it starts no worker and persists no data.
Deployments where the application role lacks schema-creation rights can use
the ledger comparison, but physical inspection reports an actionable error.

Catalog normalization strips the scratch schema qualifier from defaults and
index definitions. It compares logical object definitions, not row data,
query plans, ownership, grants, runtime parameters, or every PostgreSQL storage
property. SQLite behavior from ADR-0277 remains unchanged. MySQL remains
unsupported under ADR-0006 until a separately certified adapter and catalog
contract exist.

## Verification

PostgreSQL integration contracts build a reference from the full migration
bundle, detect unexpected table/column/index/constraint objects, detect a
column type mismatch and missing index/foreign key, and verify scratch schemas
are removed. The contract requires `HYDRO_TEST_POSTGRES_URL` and is run against
the supported PostgreSQL test environment.
