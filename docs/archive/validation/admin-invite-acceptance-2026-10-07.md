# Staff invitation acceptance validation

**Date:** 2026-10-07  
**Decision:** [ADR-0352](../architecture/0352-public-invitation-acceptance-form.md)

## Capacity and deployment

- The GET page renders a single bounded Templ form and creates no persistent
  service, worker, asset bundle, cache entry, or database connection of its own.
- POST bodies retain the existing 16 KiB bound and invite attempts use the
  existing 10-per-hour limiter. Password hashing and token consumption stay in
  the existing transaction.
- The route is registered only when the `auth` module is active. No schema or
  migration changes were made; SQLite, PostgreSQL, MySQL, and future adapter
  migration metadata are therefore unchanged.
- Rolling deployment requires all instances to serve the GET page before new
  invite links are handed out. The POST JSON contract remains backward
  compatible. Rollback requires no data repair.
- The form and POST responses use `Cache-Control: no-store` and
  `Referrer-Policy: no-referrer`; the page removes the token from browser
  history after rendering.

## Evidence

- `go test ./modules/designsystem ./internal/auth ./internal/auth/sqlstore ./cmd/hydro`
- Chromium E2E seeds a real invitation in the test SQLite database, loads its
  link, checks accessible form and mismatch recovery, accepts the invite, then
  signs in as the newly created editor.
- axe-core WCAG 2.1 A/AA runs on the rendered acceptance page.
- Existing invite store tests cover token single-use and tenant isolation;
  existing migration metadata covers supported dialects. No DB contract was
  modified by this page.
