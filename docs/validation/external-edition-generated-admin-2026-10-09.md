# Independent Edition generated Admin contract — 2026-10-09

The separate `tests/external-edition` Go module was revalidated after the
generated Admin validation and route-test changes.

```sh
GOCACHE="$PWD/.cache/go-build" GOTMPDIR=/tmp TMPDIR=/tmp GOPROXY=off \
  go test ./... -count=1 -p 1 -timeout=5m
GOCACHE="$PWD/.cache/go-build" GOTMPDIR=/tmp TMPDIR=/tmp GOPROXY=off \
  go vet ./...
```

Both commands passed. The contract tests cover:

- explicit generated Admin selection and 404 for an inactive resource;
- the generated SQLite migration, tenant-resolver preflight, and absence of
  partially registered routes when that resolver is missing;
- tenant-scoped create/read and Admin list isolation between two tenants;
- generated Admin create/update/delete, missing-CSRF rejection, viewer read
  and mutation denials, editor-role shell identity and valid-CSRF
  create/update/delete with SQLite persisted-state readback, and persisted-state
  checks after denied mutations.

The Edition is a separate module and imports the public `extension/v1` Admin
contract. These tests use an in-memory HTTP mux and fixture-provided identity,
authorization, tenant, and CSRF callbacks. They do not certify production
session middleware, real browser behavior, PostgreSQL, or deployment setup.
The callback boundary remains Edition-owned as required by ADR-0349/0354/0355.

## Editor and Viewer browser journey added

ADR-0365 adds `PageRoleViewer` to the public v1 renderer contract. The external
test server accepts test-only `admin`, `editor`, and `viewer` identity headers.
The shared shell renders Viewer with read-only capabilities; the resource API
allows reads for all three roles and writes only for admin/editor. A Playwright
scenario exercises editor list/create/edit/delete, Viewer list/detail, hidden
CRUD affordances and privileged shell navigation, valid-CSRF
create/update/delete denial, direct API
create/update/delete denial, persistence checks, axe, and 390px overflow. The
nested SQLite contract proves Viewer list/detail reads, all six Admin/API
write denials, and unchanged persistence. `go test ./...` and `go vet ./...`
in this independent module passed after expanding the mutation matrix.
Playwright discovered the test
(`--list`: 1/1). The actual browser scenario has not run here because the
runner cannot bind the Edition test server; it needs the external fixture below
in a loopback-capable test environment:

```sh
# Terminal 1, from tests/external-edition
go run ./cmd/testserver --dsn 'file:/tmp/external-edition-e2e.sqlite?mode=rwc'

# Terminal 2, from e2e
HYDRO_E2E_EXTERNAL_EDITION=1 \
HYDRO_E2E_BASE_URL=http://127.0.0.1:8099 \
HYDRO_E2E_DATABASE_DSN='file:/tmp/external-edition-e2e.sqlite?mode=rwc' \
npx playwright test custom/external-edition-editor.spec.ts --project=chromium --workers=1
```

The test server uses test identity headers and a fixture policy, not production
sessions or authorization. SQLite HTTP composition tests passed after the
change; no live browser or PostgreSQL claim is made.

Local server startup was attempted with the documented SQLite DSN. The fixture
opened/migrated the database and composed the generated routes before
`net.Listen` failed with `socket: operation not permitted` on
`127.0.0.1:8099`. This confirms the remaining E2E blocker is listener access;
it is not a Playwright assertion result.
