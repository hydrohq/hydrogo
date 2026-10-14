# ADR-0006: Initial version SQLite/PostgreSQL and extensible DB adapters

- **Status:** Accepted; MySQL deferral partially superseded by [ADR-0283](0283-mysql-mariadb-adapter.md)
- **Date:** 2026-09-05
- **Supersedes:** [ADR-0002](0002-database-portability.md)

## Context

Equally supporting every database in the initial release increases development and testing costs. Designing for only a single database, however, makes it difficult to add new engines in the future. HydroGo must initially cover the small setup and the multi-instance production setup, and be able to add new databases when real demand arises.

## Decision

The certified databases of the first stable version are:

- **SQLite:** Compact, single-node, and easily portable setup.
- **PostgreSQL:** Standard and Distributed production setups.

MySQL and other SQL engines are not in the initial scope. The architecture supports adding these later as **certified adapters**. A "supports all databases" promise is not made.

## Boundaries

### Platform adapter

The platform-level adapter provides only infrastructure behavior:

```go
type Adapter interface {
    Name() string
    Capabilities() CapabilitySet
    Open(context.Context, Config) (Pool, error)
    Begin(context.Context, TxOptions) (Tx, error)
    ClassifyError(error) ErrorKind
}
```

This interface does not try to generalize all SQL features.

### Module store

Each module defines the repository/store contract it uses. SQLite and PostgreSQL queries are implemented separately:

```text
modules/news/store/
├── contract.go
├── sqlite/
└── postgres/
```

Adding a new database adapter is not just writing a connection driver. For each supported Core component and module, the following are required:

- Dialect queries
- Migrations
- Error mapping
- Transaction and locking behavior
- Backup/restore integration
- Contract and migration tests

### Capability model

The adapter explicitly declares its capabilities:

```text
returning
json_native
full_text
skip_locked
advisory_lock
online_ddl
listen_notify
```

The module declares its required capabilities. If the selected database does not provide them, the system throws an explicit error at startup or the module uses its own portable fallback.

## Portable data rules

- Identity is a UUID/ULID generated on the application side.
- Dates are stored in UTC and with an explicit precision contract.
- Slugs, emails, and unique keys are protected by normalized helper fields; it is not left to collation assumptions.
- Money is stored as integer minor-unit and currency code.
- Pagination uses a stable sort and cursor contract.
- Domain behavior is not strictly tied to a DB-specific trigger or stored procedure.

## Migration

Each migration has selected and certified dialect implementations:

```text
migrations/<id>/
├── sqlite.sql
├── postgres.sql
└── migration.yaml
```

If a MySQL adapter is added in the future, `mysql.sql` is added to the same logical migration sequences, or the engine's initial baseline schema is generated. Old migration checksums are not changed.

## Development and testing

- Fast local development can be done with SQLite.
- If the customer's production target is PostgreSQL, CI and staging use PostgreSQL.
- No module is considered complete without passing the SQLite and PostgreSQL common contract tests.
- A new adapter is not listed as "supported" without passing the same test kit.

## Consequences

- The test matrix for the initial release remains manageable.
- Two actual topologies, small and distributed, are verified.
- The path to adding a new DB is open, but no claim of false universality is made.
- DB-specific optimizations can be used within capability boundaries.
- New DB support is still a significant product investment; the adapter interface does not make it zero-cost.
