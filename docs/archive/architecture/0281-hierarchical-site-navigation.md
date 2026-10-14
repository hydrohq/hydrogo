# ADR-0281: Hierarchical Site navigation

- **Status:** Accepted
- **Date:** 2026-09-28
- **Related:** [ADR-0172](0172-tenant-scoped-site-data.md),
  [ADR-0273](0273-optional-sectoral-content-modules.md)

## Context

Site editors need to organize localized header and footer links into nested
menus. The existing Site module stores a bounded, flat list. Navigation must
remain an optional Site capability and preserve existing editions and readers.

## Decision

Extend each localized navigation row with an optional `parent_id` and an
allowlisted link `target`. Empty parent means a root item. Sibling and root
ordering share the existing per-locale position sequence, preserving the
existing uniqueness contract. Saving a menu validates unique IDs, parent
existence, bounded depth, and acyclicity before replacing a locale's rows in
one transaction. Public navigation continues to expose a bounded flat
projection carrying parent and target fields; themes build their tree from
that stable projection.

## Compatibility and operations

- The Site-only expand migration adds columns with values that preserve all
  existing rows. Old binaries ignore the columns; rolling back the migration
  removes only hierarchy metadata and keeps labels and links.
- SQLite and PostgreSQL use the same string contract; no dialect detail enters
  the domain service.
- No worker, cache, or additional connection is introduced. The existing
  500-row public bound and Site module activation gate remain in place.
- Menu edits remain tenant and locale scoped and are written through the
  existing transaction and audit boundary.
