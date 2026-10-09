# Generated Admin service-error browser feedback — 2026-10-09

## Scope and result

The independent Edition browser journey injects a test-only 503 at its
generated resource API boundary during a valid Admin create. Chromium verifies
that the Admin responds with a user-facing 502 form, identifies the failed
save and offers a retry message, preserves the entered title, and does not
persist the record. The test then removes the injected failure, retries the
same form, and verifies the record is created and read back through the API.
The injection header is honored only by the loopback test server; no product
route or customer configuration was added.

The full `custom/external-edition-editor.spec.ts` file passed **2/2** on
SQLite and **2/2** on PostgreSQL 18. Besides the new save-error/retry journey,
it covers editor/viewer CRUD boundaries, validation, record-level policy,
tenant media selection, persisted state, axe checks, and mobile overflow. The
separate Edition module `go test ./... -count=1 -p 1 -timeout=5m` and `go vet
./...` passed after the fixture change.

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

## Limits

The injected outage covers a 503 during generated collection creation. It does
not simulate every API status, database outage/recovery, or failure while
editing, deleting, loading relation options, or serving editorial workflows.
