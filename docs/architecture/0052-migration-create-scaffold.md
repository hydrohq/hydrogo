# ADR-0052: Reviewable migration scaffold command

- **Status:** Accepted
- **Date:** 2026-09-14
- **Related decisions:** [ADR-0010](0010-sdk-lifecycle-and-edition-manifest.md), [ADR-0039](0039-conditional-cross-module-migrations.md)

## Decision

`hydro migration create` creates a new migration directory with SQLite and
PostgreSQL SQL placeholders plus the metadata required by migrationguard. It
requires an explicit timestamped identifier, refuses to overwrite an existing
directory, and marks checksums for replacement after SQL review. It never
pretends placeholder SQL is production-safe.
