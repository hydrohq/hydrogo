# Generated editorial mutation role matrix — 2026-10-09

The generated Admin HTTP handler now has an explicit role × valid-CSRF matrix
for draft save, workflow transition, and draft discard. The Edition fixture
authorizes all three roles at the outer route boundary, then grants the
editorial capability callbacks only to `admin`. This confirms hidden UI or
route-level authentication is not the authorization boundary: Editor and
Viewer direct POSTs with valid CSRF receive 403, and no persistence callback
is called. Admin receives the success redirect and each callback is called.

```sh
GOCACHE=/tmp/hydrogo-admin-test-cache GOMAXPROCS=1 GOPROXY=off \
  go test ./internal/adminstudio -run 'TestGeneratedEditorialMutationsRespectEditorAndViewerCapabilities|TestGeneratedAdminDraftsStayIsolatedAndUseTwoVersionCAS' -count=1 -p 1
GOCACHE=/tmp/hydrogo-admin-test-cache GOMAXPROCS=1 GOPROXY=off \
  go test ./internal/adminstudio -count=1 -p 1
GOCACHE=/tmp/hydrogo-admin-test-cache GOMAXPROCS=1 GOPROXY=off \
  go vet ./internal/adminstudio
```

Revision restore is covered by the same role policy contract. Admin's valid
CSRF restore succeeds, Editor and Viewer valid-CSRF restores return 403 without
invoking the restore callback, and missing CSRF or a stale resource version is
rejected before restore. Its existing history/diff test still checks escaped
snapshot content and expected-version handling.

Localized field save and cross-locale copy now use the same role matrix.
`ResourceAccess.Update` and `CanCopyLocale` grant Admin only; Editor and Viewer
valid-CSRF requests for both routes return 403, while the stored translation
and successful Admin copy state remain unchanged. The existing test also
verifies that only localized fields are sent to the Edition store.

## Basic generated CRUD

The direct Admin route test also checks role-specific capabilities for create,
update, and delete with valid CSRF:

| Role | Create | Update | Delete |
| --- | --- | --- | --- |
| Admin | allowed | allowed | allowed |
| Editor | allowed | allowed | denied |
| Viewer | denied | denied | denied |

Allowed requests reach the corresponding fake resource API exactly once and
redirect after success; denied requests return 403 without reaching it. The
existing `TestGeneratedAdminMutationUsesCSRFAndResourceAPI` independently
checks that missing/invalid CSRF is rejected before the API for basic mutation
routes. These are handler contract checks; the independent Edition SQLite and
Chromium tests cover persisted editor CRUD, while Viewer persisted-state
denial remains separately exercised by the external Edition contract.

```sh
GOCACHE=/tmp/hydrogo-admin-test-cache GOMAXPROCS=1 GOPROXY=off \
  go test ./internal/adminstudio -run '^TestGeneratedAdminRevisionHistoryDiffAndRestoreAreBounded$' -count=1 -p 1
GOCACHE=/tmp/hydrogo-admin-test-cache GOMAXPROCS=1 GOPROXY=off \
  go test ./internal/adminstudio -run '^TestGeneratedBasicCRUDUsesEditionRoleCapabilities$' -count=1 -p 1
```

All commands passed. Existing tests retain the missing-CSRF, stale-version,
and callback-denial cases. This closes the shared handler contract for draft
save, workflow transition, draft discard, and revision restore; it does not
claim real-session browser coverage or Edition database persistence for each
role. Edition callbacks remain the source of role policy.

## Draft inbox read-policy boundary

The draft-isolation handler test now additionally opens the Drafts deep link as a
Viewer whose Edition `CanReadResource` policy denies the collection. The route
returns 404 and the `ListDrafts` storage callback is not called, proving draft
metadata does not leak through a direct inbox URL. Admin/editor allow-path
rendering remains covered by the same handler test; this is a callback contract,
not production-session or browser evidence.

```sh
GOCACHE=/tmp/hydro-cover-cache go test ./internal/adminstudio \
  -run '^TestGeneratedAdminDraftsStayIsolatedAndUseTwoVersionCAS$' -count=1 -p 1
GOCACHE=/tmp/hydro-cover-cache go vet ./internal/adminstudio
```

Both passed.

## Collection denial covers every editorial deep link — 2026-10-10

A generated collection with revisions, localization, drafts, and workflow now
has a route-level read-denial matrix for 11 direct list/create/detail, draft,
revision, locale-save/copy, transition, and discard GET/POST paths. With the
Edition's collection `CanReadResource` callback denying access, every route
returns a concealed 404 before resource API, revision/draft/locale storage,
CSRF, or page rendering is reached. The callbacks are instrumented so the test
fails if any downstream boundary runs.

```sh
GOCACHE=/tmp/hydro-cover-cache go test ./internal/adminstudio \
  -run '^TestGeneratedAdminReadPolicyBlocksEveryEditorialDeepLinkBeforeStorage$' \
  -count=1 -p 1
GOCACHE=/tmp/hydro-cover-cache go test ./internal/adminstudio ./internal/i18n \
  -count=1 -p 1
GOCACHE=/tmp/hydro-cover-cache go vet ./internal/adminstudio ./internal/i18n
```

All passed. This proves the shared-handler collection deny boundary, not
production authentication middleware or resource-level policy behavior in a
deployed Edition.

## Unknown-role direct mutation denial

The basic CRUD and editorial valid-CSRF direct-handler matrices now include an
`unknown` role that the outer `Authorize` callback deliberately admits so the
Edition capability callbacks are the tested boundary. It has no Create,
Update, Delete, draft-save, transition or discard grant. Every mutation returns
403 and no fake resource API or editorial persistence callback is reached.

```sh
GOCACHE=/tmp/hydro-cover-cache go test ./internal/adminstudio \
  -run '^(TestGeneratedBasicCRUDUsesEditionRoleCapabilities|TestGeneratedEditorialMutationsRespectEditorAndViewerCapabilities)$' \
  -count=1 -p 1
```

Passed. This isolates the Edition capability contract; unknown production
identity/session issuance remains part of the separate external Edition gate.

## Draft inbox denied read roles — 2026-10-10

The direct Drafts route denial now covers Edition `CanReadResource` denying an
Editor, Viewer, and an active Admin with an explicit read denial. All receive
404 and `ListDrafts` is not invoked, so draft metadata cannot leak through the
collection deep link. This tests the Edition callback decision in the generated
handler, not the production identity middleware or browser shell.

```sh
GOCACHE=/tmp/hydro-cover-cache go test ./internal/adminstudio \
  -run '^TestGeneratedAdminDraftsStayIsolatedAndUseTwoVersionCAS$' -count=1 -p 1
GOCACHE=/tmp/hydro-cover-cache go vet ./internal/adminstudio
```

Both passed.
