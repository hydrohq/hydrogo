# Viewer News workflow denial matrix — 2026-10-09

The authenticated Editor creates a draft News item in the browser. The Viewer
can open its read-only detail. With the Viewer session’s valid CSRF token,
update, schedule, publish, and retract requests all return 403. A subsequent
read confirms the original title remains and the record is still a draft. The
read-only detail also passes axe-core WCAG 2.1 A/AA with zero violations.

The complete focused Chromium role journey passed **1/1 on SQLite** and
**1/1 on PostgreSQL 18.6**. PostgreSQL used the local `arda` role, verified as
`NOSUPERUSER NOBYPASSRLS`, against a disposable database that was dropped
after the run. The News transport and UI package tests and vet passed:

```sh
go test ./modules/news/transport ./internal/ui -count=1 -p 1 -timeout=5m
go vet ./modules/news/transport ./internal/ui
```

The Admin role journey’s permission-form helper now preserves an already-open
`details` disclosure and asserts the form remains open after selecting a
permission, avoiding a hidden submit control in this long browser scenario.

This closes only Viewer denials for News update, schedule, publish, and
retract, plus persisted draft-state verification. Taxonomy, preview, News API
write paths, full override combinations, and production Edition identity
remain outside this slice.
