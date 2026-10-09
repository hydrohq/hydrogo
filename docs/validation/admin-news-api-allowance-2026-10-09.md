# Admin News API allow-path — 2026-10-09

The real Edition Admin session requests `/csrf-token` and confirms its token
matches the `hydro_csrf` cookie. A News API create with a missing token and a
create with a mismatched token each return `403`. With the valid
`X-CSRF-Token`, the Admin creates a draft and updates it using the current
version. The response and a subsequent authenticated API read confirm the
updated title, draft status, and incremented version were persisted. With the
current version, the same Admin schedules the item, publishes it, and retracts
it. Each response increments the version and reports the expected status; a
final API read confirms the retracted state and updated content persisted.

The focused Chromium role journey passed **1/1 on SQLite** and **1/1 on
PostgreSQL 18.6**. PostgreSQL used the local `arda` role (`NOSUPERUSER
NOBYPASSRLS`) against a disposable database that was removed after the run.

This closes the Admin News API create/update/schedule/publish/retract
allow-path and create missing/mismatched CSRF cells only. Taxonomy writes,
other module routes, full persisted-state mutation coverage, and production
Edition identity remain open.
