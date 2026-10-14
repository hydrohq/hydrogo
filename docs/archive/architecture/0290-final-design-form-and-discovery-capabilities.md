# ADR-0290: Final design, form, and discovery capabilities

**Status:** Accepted  
**Date:** 2026-09-29

## Context

HydroGo's remaining reusable page blocks and form enhancements need to remain
small server-rendered contracts. Content discovery also needs deterministic
SQLite full-text search, scheduled expiry for news, and reusable local-business
metadata. The implementation must preserve optional-module behavior and work
without a frontend runtime or external image/document toolchain.

## Decision

- Page blocks remain typed Templ components and validated JSON block contracts.
- Form phone masks, counters, cascading selects, wizard state, and successful
  submission reset are progressive browser enhancements. Server-side field
  validation remains authoritative. The Turkey province/district selector uses
  an embedded MIT-licensed 81-province/922-district data snapshot and validates
  the selected pair on submission.
- SQLite FTS5 is used when the virtual table is installed by the search
  migration. The existing bounded SQL search remains the fallback. PostgreSQL
  and MySQL keep the current provider implementation.
- News expiry is an optional UTC timestamp. Public news queries exclude expired
  rows; no scheduled worker or archival mutation is required.
- LocalBusiness JSON-LD is derived from the existing site profile and emitted
  only when that profile is configured.
- The media temp sweeper is started only when the media module is active. It
  scans only the configured media root, does not follow links, and removes
  Hydro-owned temporary filename patterns after at least 24 hours.

## Compatibility and operations

The search and news changes add nullable columns/indexes through the versioned
SQLite/PostgreSQL/MySQL migration metadata. Expiry is nullable, so old content
remains visible; mixed-version deployments tolerate the additive schema. The
FTS5 virtual table is SQLite-specific and is selected only for SQLite. The
other dialect migrations are no-op compatibility records. Rollback removes
the FTS table/triggers and the nullable expiry column; operators must retain
normal database backup/recovery safeguards before rollback.

The FTS index is maintained by SQLite triggers and grows with indexed content;
the search migration performs a one-time backfill. The stale-file sweep has a
fixed six-hour schedule and bounded, immediate-directory scan. It is absent
from inactive media and CGI runtimes. No additional persistent connection,
cache, or worker pool is created.

## Verification

SQLite FTS behavior and news repository compatibility are covered by package
tests. Form contracts validate cross-field location choices. The stale-file
sweeper tests cover retention, unrelated assets, and symbolic links. Full
verification is `templ generate && go test ./...`.
