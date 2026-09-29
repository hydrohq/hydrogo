# ADR-0322: Staff Broadcast Notices

- **Status:** Accepted
- **Related decisions:** ADR-0013, ADR-0286

## Decision

The optional Site module stores a single staff broadcast notice in its
existing private tenant-scoped settings store. Notices carry plain text,
severity (`info`, `warning`, or `critical`), active state, and a required UTC
expiry. Admin Studio resolves the notice for HTML requests under `/admin` and
renders it through a typed Design System component. Staff can dismiss a notice
in the current browser; changing its content, severity, or expiry gives it a
new dismissal key. Expired or inactive notices stop rendering automatically.

## Compatibility and capacity

This adds no table, migration, worker, route, or resource when the Site module
is inactive. Existing SQLite, PostgreSQL, and MySQL schemas are unchanged.
Notice reads are limited to Admin Studio HTML GET requests and remain tenant
scoped. A database read error omits the banner and does not fail the page.

## Verification

Tests cover private setting validation, UTC expiry, dismissal behavior,
resolver request bounds, escaped Templ content, and the full package suite.
