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

```sh
GOCACHE=/tmp/hydrogo-admin-test-cache GOMAXPROCS=1 GOPROXY=off \
  go test ./internal/adminstudio -run '^TestGeneratedAdminRevisionHistoryDiffAndRestoreAreBounded$' -count=1 -p 1
```

All commands passed. Existing tests retain the missing-CSRF, stale-version,
and callback-denial cases. This closes the shared handler contract for draft
save, workflow transition, draft discard, and revision restore; it does not
claim real-session browser coverage or Edition database persistence for each
role. Edition callbacks remain the source of role policy.
