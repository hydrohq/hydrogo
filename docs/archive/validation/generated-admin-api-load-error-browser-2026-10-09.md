# Generated Admin API read-error recovery — 2026-10-09

## Behavior and verification

Generated Admin API read failures now return a localized, server-rendered
Studio page with an accessible `role="alert"` and a typed GET retry form to the
current resource path. Native form submission ensures retries still work when
the browser is already at that same URL; the Studio SPA intentionally ignores
same-URL anchor navigation. The response keeps `502` and `private, no-store`;
upstream status body, database URL, and panic details are not included in the
page.

The handler contract test verifies Turkish and English copy, response status and
headers, retry form, shell rendering, and backend-detail redaction. The
independent Edition Chromium journey injects read-only 503s during collection
list and record detail loading. It checks the alert, retry path and axe WCAG
2.1 A/AA, removes each fault, then retries and sees the record. The complete
editor/media browser file passed 3/3 on SQLite and 3/3 on PostgreSQL 18. The
independent Edition Go module tests/vet and `internal/adminstudio` plus
`internal/i18n` tests/vet passed.

The test-only Edition server has a dedicated `X-Test-Generated-API-Read-Status`
fault header for GETs; the write-failure header remains limited to mutations.
Both use disposable local databases and fixture identity/policy callbacks.

```sh
go test ./internal/adminstudio/... ./internal/i18n/... -count=1
go vet ./internal/adminstudio/... ./internal/i18n/...
cd tests/external-edition && go test ./... -count=1 -p 1 -timeout=5m && go vet ./...
cd ../.. && go test ./... -count=1 -p 1 -timeout=8m && go vet ./...
go run ./tools/moduleguard -root modules -allow modules/api -dependencies modules/dependencies.json
go run ./tools/migrationguard -root migrations
go run ./tools/importguard -root examples/client-edition
go run ./tools/importguard -root examples/news-edition
go run ./tools/importguard -root examples/construction-edition

cd e2e
HYDRO_E2E_EXTERNAL_EDITION=1 HYDRO_E2E_BASE_URL=http://127.0.0.1:<port> \
  HYDRO_E2E_DATABASE_DSN='<disposable-sqlite-dsn>' \
  npx playwright test custom/external-edition-editor.spec.ts \
  --project=chromium --workers=1 --reporter=line --trace=off

HYDRO_E2E_EXTERNAL_EDITION=1 HYDRO_E2E_BASE_URL=http://127.0.0.1:<port> \
  HYDRO_E2E_DATABASE_DIALECT=postgres \
  HYDRO_E2E_DATABASE_DSN='<disposable-postgres-dsn>' \
  npx playwright test custom/external-edition-editor.spec.ts \
  --project=chromium --workers=1 --reporter=line --trace=off
```

This certifies generated collection list and record-detail read-error flows
against the independent Edition fixture. It does not simulate a database
outage, relation/editorial reads, a production identity callback, or a live
customer deployment.

The full root Go run had no PostgreSQL DSN; PostgreSQL evidence here is the
separately run Chromium 18.6 journey above.
