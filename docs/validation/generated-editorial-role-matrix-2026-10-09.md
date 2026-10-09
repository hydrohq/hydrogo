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
