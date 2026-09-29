# ADR-0317: Custom public 404 and 500 pages

## Status

Accepted

## Decision

When the optional Site module is enabled, Site managers may edit public 404
and 500 copy, an illustration path, an internal call-to-action, and up to
eight recommended internal links. Settings use the existing tenant-scoped
`site_settings` contract; no schema migration or new worker is needed.
Illustrations must use local `/media/` paths and all links must remain on the
current site. Invalid or stale stored values fall back to friendly defaults.

The public HTML response layer replaces 404 and 500 bodies for public page
routes while preserving their status codes. It excludes API, authentication,
health, and admin route families so their response contracts remain intact.
The same layer handles normalized server errors and recovered panics. Output
uses typed Design System Templ components with escaped copy and validated
URLs, and carries `Cache-Control: no-store`. The error editor requires Site
management authorization and uses the standard CSRF-protected admin routes.

The response wrapper buffers only bounded error bodies and passes successful
responses through. The feature adds no runtime work when Site is disabled.
SQLite/PostgreSQL/MySQL behavior is unchanged because the current settings
store is reused. Existing binaries continue to ignore the new settings; a
rollback removes only the editor/routes and leaves harmless settings rows.

## Verification

Tests cover setting validation and fallback, editor authorization/save,
rendered 404/500 status and copy, API exclusion, and friendly panic recovery.
