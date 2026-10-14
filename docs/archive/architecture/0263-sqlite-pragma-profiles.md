# ADR-0263: SQLite PRAGMA profiles

- **Status:** Accepted
- **Date:** 2026-09-27

## Context

SQLite connection defaults were fixed for every capacity profile. Compact
hosting needs a small cache footprint, while VPS deployments can trade bounded
per-connection memory and a longer lock wait for throughput.

## Decision

SQLite connections receive one of two DSN-backed PRAGMA profiles. Compact
keeps a 2 MiB page cache and 5 second busy timeout. VPS uses an 8 MiB cache,
64 MiB memory map, and 10 second busy timeout. Both retain foreign keys, WAL,
and `synchronous=NORMAL`. HydroGo maps its Compact capacity profile to
Compact; Standard and Distributed use VPS. Direct callers default to Compact.

These are SQLite adapter settings only. Application services continue to use
the provider-neutral database contract. PostgreSQL behavior is unchanged.

## Consequences

- No migration or rollback is required; reverting the selected profile changes
  settings on the next connection.
- The VPS cache is bounded per connection. Existing pool limits continue to
  bound aggregate memory; memory-mapped pages are not an eager allocation.
- Both profiles retain WAL recovery and existing transaction semantics.
- Contract tests inspect the active PRAGMA values through SQLite connections.
