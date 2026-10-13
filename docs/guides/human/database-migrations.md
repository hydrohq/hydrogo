# Database Migrations (Human Guide)

A "migration" is a recorded, versioned change to your database schema (tables, columns, indexes). In HydroGo, each migration lives in its own directory and contains **separate SQL files for each supported database dialect** (SQLite, PostgreSQL, MySQL) — because these three engines have subtle but critical differences in their SQL dialects.

## How to Run Migrations

```sh
hydro migrate --config config.yaml
```

This command is idempotent and **safe to re-run repeatedly** — it automatically skips already applied migrations and executes only pending ones. In local development (`hydro run`), migrations run automatically on startup.

## Architectural Lesson: Dialect Parity

Across database-backed applications, a common pitfall is treating queries as generic ANSI SQL, testing only on PostgreSQL or SQLite, and neglecting MySQL/MariaDB syntax (or vice versa). In HydroGo, all database operations are explicitly certified across SQLite, PostgreSQL, and MySQL.

If you ever encounter an issue where SQLite functions properly but MySQL throws syntax or collation errors, verify dialect-specific queries and column definitions. All repositories must use dialect-aware queries rather than assuming PostgreSQL or SQLite defaults.

## Adding a New Migration (For Developers)

Every migration directory contains a `migration.yaml` metadata file and a **SHA-256 checksum** for each dialect's SQL file. If you modify a migration file prior to releasing it, you must recompute its checksum — otherwise, the automated verification tools (`migrationguard`) will flag a checksum mismatch and prevent CI/CD passage.

## Migrations Are Immutable

Once a migration has been applied or published, its SQL file must **never be modified in place**. If schema alterations are required, write a new, sequential migration file. This preserves history and schema determinism across all deployment environments.
