# ADR-0030: Bounded admin list query contract

## Status

Accepted

## Decision

Admin list surfaces use a shared `ListQuery` with search, named filters,
explicit sort, and bounded page parameters. Parsing is centralized at the
presentation boundary; modules translate the result into repository-specific
predicates. SQL dialects and search engines remain adapter details.

## Compatibility and capacity

The contract is additive, SQLite/PostgreSQL neutral, and requires no migration.
Page size is capped at 100 to prevent accidental unbounded reads. Invalid or
missing values normalize to safe defaults, with no workers or cache state added.
