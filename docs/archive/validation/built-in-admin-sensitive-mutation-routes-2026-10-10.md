# Built-in Admin sensitive route coverage — 2026-10-10

The built-in mutation CSRF matrix did not include the composition-root routes
for backup creation and IP ban create/remove. Production now registers those
routes through the same small route builders the test consumes. The existing
route matrix verifies each is `router.Protected`, rejects absent and forged
CSRF before handler dispatch, rejects anonymous/inactive sessions, and
dispatches with valid Admin session plus CSRF.

Extracted the identical active-platform-Admin predicate shared by the backup,
security ban, audit, and extension Admin handlers. Its table test covers active
Admin, inactive Admin, editor, viewer, anonymous, and Admin permission-overrides
that must not demote the platform role.

Validation passed:

```sh
GOCACHE=.cache/adminstudio-gocache GOTMPDIR=.cache/adminstudio-gotmp \
  TMPDIR=.cache/adminstudio-tmp go test ./cmd/hydro \
  -run 'TestBuiltInMutationRoutesAreProtectedAndCSRFChecked|TestActivePlatformAdminRoleBoundary' \
  -count=1 -p 1
GOCACHE=.cache/adminstudio-gocache GOTMPDIR=.cache/adminstudio-gotmp \
  TMPDIR=.cache/adminstudio-tmp go vet ./cmd/hydro
```

This closes the sensitive built-in mutation route registration gap for backup
and IP bans and verifies the shared role predicate. It does not complete every
Admin route's persisted-state behavior, browser journey, or SQLite/PostgreSQL
parity matrix.
