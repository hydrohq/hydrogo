# Generated Admin service-error browser feedback — 2026-10-09

## Scope and result

The independent Edition browser journey injects a test-only 503 at its
generated resource API boundary during valid Admin create and edit requests.
For both operations Chromium verifies a user-facing 502 form, a plain-language
retry message, and retained input. Create failure leaves no record; edit
failure leaves the persisted record unchanged. Removing the injected failure
and retrying creates/updates the record, which is then read back through the
API. The injection header is honored only by the loopback test server; no
product route or customer configuration was added.

The full `custom/external-edition-editor.spec.ts` file passed **2/2** on
SQLite and **2/2** on PostgreSQL 18. Besides create/edit save-error and retry,
it covers editor/viewer CRUD boundaries, validation, record-level policy,
tenant media selection, persisted state, axe checks, and mobile overflow. The
separate Edition module `go test ./... -count=1 -p 1 -timeout=5m` and `go vet
./...` passed.

Commands:

```sh
# Start tests/external-edition/cmd/testserver on loopback with SQLite, then:
HYDRO_E2E_EXTERNAL_EDITION=1 \
HYDRO_E2E_BASE_URL=http://127.0.0.1:<port> \
HYDRO_E2E_DATABASE_DSN='file:/tmp/external-edition-e2e.sqlite?mode=rwc' \
npm test -- --project=chromium --workers=1 custom/external-edition-editor.spec.ts

# PostgreSQL 18 uses the same command with a PostgreSQL DSN and a server
# started using --dialect postgres.
```

Both browser runs used disposable local databases and a fixture-provided
identity/policy boundary. They do not certify production sessions, customer
Edition authorization callbacks, deployment, or Core Media storage.

## Delete failure and retry — 2026-10-09

The browser journey now injects a 503 only for generated API mutations, leaving
GET available for the Admin to reload the record after a failed delete. Chromium
verifies the localized delete error is visible and accessible, the delete form
is restored, and the record remains persisted. After removing the injected
fault, retry deletes the record and the API returns 404. The updated complete
file passed 2/2 on SQLite and 2/2 on PostgreSQL 18. The external Edition Go
module tests and vet passed.

```sh
# With tests/external-edition/cmd/testserver running on the matching disposable DB:
HYDRO_E2E_EXTERNAL_EDITION=1 HYDRO_E2E_BASE_URL=http://127.0.0.1:<port> \
  HYDRO_E2E_DATABASE_DSN='<matching-sqlite-or-postgres-dsn>' \
  npx playwright test custom/external-edition-editor.spec.ts \
  --project=chromium --workers=1 --reporter=line --trace=off
```

## Limits

The test-server fault injector is test-only and now applies only to writes.
This is a fixture-provided identity/policy and disposable database journey; it
does not simulate every API status, database outage/recovery, relation-option
loading failure, production runtime outage, or production Edition callbacks.
