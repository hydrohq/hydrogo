# Viewer news read/write boundary — 2026-10-09

The real Admin-management Chromium journey now checks the Viewer’s allowed
content-read path as well as the denied write path. From the Viewer’s page
list, the shared sidebar exposes both Pages and News. `/admin/news` returns
200, while the “new news” action is absent. A create request carrying the
session’s valid CSRF token returns 403, and a subsequent database-backed news
list does not contain the attempted title.

The focused `editor and viewer sessions enforce the Edition role boundary`
Playwright scenario passed **1/1 on SQLite** and **1/1 on PostgreSQL 18.6**.
The PostgreSQL E2E used the local `arda` role, which is `NOSUPERUSER`
`NOBYPASSRLS`; the disposable database was dropped after the run. The
following Go checks also passed:

```sh
go test ./modules/news/transport ./internal/ui -count=1 -p 1 -timeout=5m
go vet ./modules/news/transport ./internal/ui
```

This closes only the default Viewer news-list read and valid-CSRF create-deny
browser cells. News update/publish/schedule, explicit content-read/write
overrides across every News route, module API mutations, deployed Edition
identity, and the full Admin role/route matrix remain open.
