# ADR-0238: Enforce Global Singleton Identity at the Database Boundary

- **Status:** Accepted
- **Date:** 2026-09-22
- **Scope:** Generated global resource migrations
- **Related:** ADR-0224 (versioned global resource contract)

## Decision

Generated migrations for `type: global` resources include an explicit `id = 'global'` CHECK constraint. The generated validation and transport layers normalize this constant identifier, while the database-level constraint prevents direct SQL queries or adapter misuse from creating multiple singleton rows.

The constraint represents a provider-neutral invariant supported across both SQLite and PostgreSQL. Generated migrations remain reviewable artifacts; DDL is never executed implicitly at runtime. Applying this constraint to existing production tables follows standard expand/migrate procedures and deduplication by operators.

## Consequences

- The stable singleton ID contract of global transports is enforced at the database storage boundary.
- Dialect parity and consistency between SQLite and PostgreSQL are strengthened.
- Existing records with invalid IDs fail closed during migration, alerting operators to remediate data prior to schema enforcement.
