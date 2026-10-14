# Generated Admin CSRF route inventory — 2026-10-10

Compared the POST dispatch branches in `GeneratedAdminHandler.ServeHTTP`
against `TestGeneratedAdminAllMutationRoutesRejectInvalidCSRFBeforePersistence`.
The test already covered create, update, delete, workflow transition, revision
restore, locale save/copy, draft save, and draft discard. The separately
supported `/admin/resources/{collection}/new/draft` create alias was missing;
it now runs the same missing-token and forged-token checks. Each case asserts
HTTP 403 and verifies both API and Edition persistence callbacks remain
untouched.

Validation passed:

```sh
GOCACHE=.cache/adminstudio-gocache GOTMPDIR=.cache/adminstudio-gotmp \
  TMPDIR=.cache/adminstudio-tmp go test ./internal/adminstudio \
  -run TestGeneratedAdminAllMutationRoutesRejectInvalidCSRFBeforePersistence -count=1
GOCACHE=.cache/adminstudio-gocache GOTMPDIR=.cache/adminstudio-gotmp \
  TMPDIR=.cache/adminstudio-tmp go vet ./internal/adminstudio
```

This closes the generated-handler POST alias gap only. It does not complete the
broader built-in Admin role/route/CSRF matrix or SQLite/PostgreSQL comparison.
