# ADR-0002: SQLite, MySQL and PostgreSQL support

- **Status:** Superseded — [ADR-0006](0006-extensible-database-adapters.md)
- **Date:** 2026-09-05

## Context

HydroGo must work from a small VPS with 512 MB RAM to high-traffic and multi-node installations. A single database choice either unnecessarily burdens small installations or limits large installations.

## Decision

HydroGo provides three first-class database profiles:

| Driver | Main use | Deployment limit |
|---|---|---|
| SQLite | Nano/Standalone, single node | Local persistent disk, single active application node |
| PostgreSQL | Scale-out and high concurrency | Multiple HydroGo instances, managed or separate DB |
| MySQL | Enterprises with MySQL standard | InnoDB, supported version matrix |

`psql` is PostgreSQL's command-line tool; the architectural driver name will be `postgres`.

SQLite is the initial reference implementation. PostgreSQL and MySQL adapters must pass the same repository contract tests before the stable `v1` release.

## Design rules

### Domain and service layer

- Domain and application service code are unaware of the SQL dialect.
- Repository interfaces are defined by the module that uses them.
- Transaction boundary is requested by the application service, applied to the driver by the platform.
- Database-specific error codes are translated into common `conflict`, `not_found`, `unavailable`, and `retryable` classes.

### Physical schema

- Each dialect uses standard tables and indexes; EAV main data model is not used.
- IDs are portable UUID/ULID generated on the application side.
- Dates are stored in UTC.
- Money is stored with integer minor-unit and currency; floating point is not used.
- Boolean, JSON, full-text, generated column, and collation differences are treated as capabilities.
- Core functionality is not strictly tied to the specific feature of a single database.

### Queries and code generation

Common behavior remains in a single service contract; when SQL is needed, it is written separately according to the dialect:

```text
modules/news/store/sqlite/
modules/news/store/postgres/
modules/news/store/mysql/
```

`sqlc` is an evaluation candidate; current documentation describes MySQL and PostgreSQL support for Go as stable, and SQLite support as beta. Therefore, the choice is not a permanent architectural rule until the three-driver prototype and benchmark are completed.

### Migration

Each logical migration can have three dialect implementations:

```text
migrations/202609050001_create_articles/
├── sqlite.sql
├── postgres.sql
├── mysql.sql
└── migration.yaml
```

`migration.yaml` carries the following:

- Migration ID and checksum
- `online`, `backfill`, or `maintenance` class
- Backward compatible binary range
- Estimated lock and disk impact
- Rollback strategy

Once published, a migration is not changed.

## Test matrix

For each module, the same behavior tests run on this matrix:

```text
SQLite × current
PostgreSQL × supported major versions
MySQL/InnoDB × supported major versions
```

Tests are not limited to CRUD; they include optimistic locking, isolation, unique conflict, rollback, deadlock retry, pagination, and migration scenarios. Since PostgreSQL and MySQL have different concurrency/locking behaviors, just running the same SQL is not considered equal support.

## Backup

- SQLite: Online Backup API or verified `VACUUM INTO`, along with an immutable media manifest.
- PostgreSQL: Provider's snapshot/PITR system or PostgreSQL compatible tools.
- MySQL: Provider's snapshot/PITR system or MySQL compatible tools.

HydroGo orchestrates the reliable backup method of each engine; it does not apply a single file copy approach to all engines.

## Consequences

- Small installations do not need an external DB service.
- Large installations can scale horizontally.
- Three dialects significantly increase development and CI costs.
- A new module is not considered "first-class" until it is completed for the three drivers.
- Performance is not sacrificed for the lowest common denominator; capabilities and driver-specific optimized queries can be used.

## References

- [PostgreSQL concurrency control](https://www.postgresql.org/docs/current/mvcc.html)
- [MySQL InnoDB transaction model](https://dev.mysql.com/doc/refman/8.4/en/innodb-transaction-model.html)
- [sqlc database support](https://docs.sqlc.dev/en/latest/reference/language-support.html)
