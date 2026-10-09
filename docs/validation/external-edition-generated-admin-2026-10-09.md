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
Playwright discovery now finds both the editor/viewer CRUD and media picker
journeys (`--list`: 2/2). The actual browser scenarios have not run here because the
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

## Generated media field integration

The Edition now has a separate additive `media_article` Blueprint and generated
resource, leaving the existing `article` migration unchanged. The SQLite HTTP
contract test verifies that the generated form renders the shared picker,
tenant-specific fixture assets are listed, an authorized media ID persists and
reads back through the generated resource API, a forged cross-tenant reference
is rejected, and the resource itself remains tenant-isolated. The nested module's
full `go test ./... -count=1 -p 1 -timeout=5m` and `go vet ./...` passed.

This composes a synthetic Edition `/api/media` fixture and tenant-reference
guard; it does not yet compose the production Core Media service, storage, or
policy callbacks. A Playwright selection-and-save scenario is added and
discovered, but has not run because this environment cannot bind the test
server loopback listener.
PostgreSQL certification for this new resource is also pending a runner with a
working PostgreSQL listener. Do not treat the synthetic fixture as production
Media-service integration.

An in-place media field addition to the already generated Project Edition
article was rejected by `hydro generate scaffold` because its existing generated
migration is immutable. The new fixture uses a separate additive resource and
includes dialect migration metadata; it does not rewrite the applied migration.

## Required text validation and editor feedback

Blueprint-generated required `string`, `text`, and `uuid` checks now reject
values that contain only whitespace while leaving the submitted value intact
in the failed form. The external Edition SQLite contract posts a whitespace-only
article title through the registered Admin handler, verifies localized 422
field feedback, and confirms the failed record was not persisted. The opt-in
Chromium journey adds the same error state to axe validation before continuing
through editor CRUD. Generator, external Edition, and Project Edition tests and
vets passed; the browser scenario remains unrun due to the listener restriction
above.

## Edition revalidation — 2026-10-09

After adding the generated `ResourceAccess` wiring guidance, both the
independent Edition module and the native Project Edition were rerun with
`go test ./... -count=1 -p 1` and `go vet ./...`; all packages passed. The
generator's focused checklist emission test also passed. This verifies source
and generated output compatibility; it does not execute the browser journey or
exercise PostgreSQL.

The latest external Edition server start was retried on 2026-10-09. Database
migration and route composition completed, but the listener again failed at
`127.0.0.1:8099` with `socket: operation not permitted`. Playwright `--list`
passed with 2/2 scenarios using the shared local package; this remains a
discovery result, not browser execution.

The `e2e` npm dependencies were installed offline from the local cache with
`npm ci --offline --no-audit --no-fund`, and Playwright independently listed
both scenarios. A new SQLite fixture DSN again reached `net.Listen` and failed
with `socket: operation not permitted`. No browser assertions ran; repeat on a
loopback-capable runner.

## Independent Go module regression — 2026-10-09

The root module suite does not recurse into this nested Edition Go module. Its
own test suite and vet were therefore run separately after the Admin Studio
changes:

```sh
cd tests/external-edition
GOCACHE=/tmp/hydrogo-go-build GOMAXPROCS=1 GOPROXY=off TMPDIR=/tmp/hydrogo-admin-tests \
  go test ./... -count=1 -p 1 -timeout=5m
GOCACHE=/tmp/hydrogo-go-build GOMAXPROCS=1 GOPROXY=off TMPDIR=/tmp/hydrogo-admin-tests \
  go vet ./...
cd ../..
GOCACHE=/tmp/hydrogo-go-build GOMAXPROCS=1 GOPROXY=off TMPDIR=/tmp/hydrogo-admin-tests \
  go run ./tools/importguard -root tests/external-edition
```

All external module packages passed; vet and the Edition import boundary
guard exited successfully. This exercises SQLite/in-process HTTP composition;
it is not the unexecuted Playwright browser journey or PostgreSQL certification.

## Invalid Admin CSRF mutation regression — 2026-10-09

The generated Admin HTTP composition contract now rejects forged CSRF tokens
for create, update, and delete. The test also verifies that a rejected create
leaves no row, a rejected update preserves the prior title, and a rejected
delete leaves the target row readable. The complete nested Edition suite,
`go vet ./...`, and `git diff --check` passed. These results use the fixture's
SQLite database and `ValidateCSRF` callback; browser/session middleware and
PostgreSQL remain separate open gates.

The independent Edition Playwright editor journey also now attempts forged
CSRF create, update, and delete requests and checks that create leaves no row
and update/delete preserve the existing row. Playwright successfully discovers
both editor/viewer and media scenarios after the change. The actual browser
assertions remain unrun because the fixture cannot bind its loopback listener
in this environment; discovery is not a passing browser result.

## Anonymous and unknown-role route matrix — 2026-10-09

The SQLite composition contract now sends anonymous and unrecognized-role
requests through Admin index/list/create/detail/update/delete and resource API
list/detail/create/update/delete. Admin and mutation endpoints return `403`;
unreadable API record detail returns `404` to conceal record existence. The
test verifies neither role can create a row or change the existing tenant
record. The nested Edition suite and `go vet ./...` passed. This closes
fixture-contract cells only: production session identity and loopback browser
execution remain open.

## Viewer direct-create deep-link denial — 2026-10-09

The independent Edition SQLite HTTP contract now requests
`GET /admin/resources/article/new` as Viewer and requires `403`, in addition
to checking that the list omits the create link and the record detail is
read-only. The Playwright viewer journey makes the same direct request and
expects `403`. The nested Edition suite and vet passed; the importguard passed
and Playwright discovered both Edition scenarios. Browser assertions remain
unrun because the fixture server cannot bind the loopback listener here.

## Live browser revalidation and read-only detail navigation — 2026-10-09

With a loopback-capable runner, the fresh SQLite external Edition server was
started and both Playwright scenarios passed (2/2): editor create, validation,
update, delete and forged-CSRF/policy-denial checks; Viewer read-only list,
detail, hidden mutation affordances, direct denied create, API/Admin mutation
denials and persistence checks; plus tenant-scoped media selection and
cross-tenant reference rejection. The independent module suite, vet, and
importguard passed against the final source tree.

The browser replay exposed that a readable record title had no link when the
current role lacked `Update`. Generated list rows now link to the read-only
detail URL whenever the collection is readable; the explicit “Düzenle” action
remains limited to update-capable users. The HTTP contract and browser journey
both verify the detail contains no mutation form and Viewer writes remain
denied. The separate generated rich-text Edition suite passed 8/8 on Chromium,
including save failure/retry, block round-trip and this read-only navigation.

The browser server uses test-only identity headers and SQLite fixtures. These
results do not certify production session middleware or PostgreSQL for this
external Edition fixture.

## Record-level update and delete policy — 2026-10-09

The fixture now seeds an article which the editor may read but may not update
or delete. The generated Admin shows a read-only detail with no mutation forms;
direct Admin and resource API update/delete requests return 403, and API
readback proves the title is unchanged. A different article remains editable
by the same editor. The external Edition Playwright file passed 2/2 on both
SQLite and PostgreSQL 18.6. The nested Edition `go test ./...`, vet, and
`importguard` passed. This is a test Edition record-policy fixture, not a
production policy implementation; its test-only identity headers keep
production login/session integration open.

## Direct browser replay follow-up — 2026-10-09

The standalone Admin role/tenant Chromium journey passes 1/1 on SQLite; the
independent Editor/Viewer and media journeys pass 2/2 against the loopback
fixture. Replay exposed test defects: the `Articles` locator also matched
`Media articles`, and the Admin journey did not handle the delete dialog, so
Playwright dismissed the confirmation by default. The fixture keeps Viewer
collection reads enabled, matching the intended read-only model; Viewer list
and detail work while create/update/delete are denied by `ResourceAccess`.
The journeys now complete Admin and Editor CRUD, Viewer read-only navigation,
cross-tenant isolation, CSRF denial, media selection, and persisted-state
checks. The full independent Edition Go suite, vet, and importguard passed on
the final source tree. This is fixture-level SQLite browser evidence;
production authentication and PostgreSQL remain open.

```sh
cd e2e
npx playwright test --project=chromium --workers=1 custom/external-edition-admin.spec.ts \
  --reporter=line --trace=off
# Start the loopback test server on 127.0.0.1:18099 with a fresh SQLite DSN,
# then run the editor/viewer and media journeys:
HYDRO_E2E_EXTERNAL_EDITION=1 \
HYDRO_E2E_BASE_URL=http://127.0.0.1:18099 \
HYDRO_E2E_DATABASE_DSN='file:/tmp/hydro_external_editor.sqlite?mode=rwc' \
npx playwright test --project=chromium --workers=1 custom/external-edition-editor.spec.ts \
  --reporter=line --trace=off
cd ../tests/external-edition
GOCACHE=/tmp/hydrogo-go-build GOMAXPROCS=1 GOPROXY=off TMPDIR=/tmp/hydrogo-admin-tests \
  go test ./... -count=1 -p 1 -timeout=5m
GOCACHE=/tmp/hydrogo-go-build GOMAXPROCS=1 GOPROXY=off TMPDIR=/tmp/hydrogo-admin-tests \
  go vet ./...
GOCACHE=/tmp/hydrogo-go-build GOMAXPROCS=1 GOPROXY=off TMPDIR=/tmp/hydrogo-admin-tests \
  go run ../../tools/importguard -root .
```
