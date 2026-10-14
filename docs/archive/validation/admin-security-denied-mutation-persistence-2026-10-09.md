# Denied Admin security mutations preserve state — 2026-10-09

## Scope and result

The Editor/Viewer browser role journey submits valid-CSRF requests to add and
remove an IP ban and to change password-reset/invitation link lifetimes. Every
request returns 403. A separately authenticated Admin then reads the security
pages: the attempted address is absent and both token-lifetime selections
still equal the values captured before the role requests. This proves the
denial protects persisted security state, not only the response status.

The complete `custom/admin-management.spec.ts` Chromium file passed **5/5** on
SQLite (2.7 minutes) and **5/5** on PostgreSQL 18 (2.8 minutes). PostgreSQL
used a `NOSUPERUSER NOBYPASSRLS` application role and a separately provisioned
`BYPASSRLS` backup operator. Its disposable server/database were removed.

```sh
# SQLite
npm test -- --project=chromium --workers=1 custom/admin-management.spec.ts

# PostgreSQL 18
HYDRO_E2E_DATABASE_DIALECT=postgres \
HYDRO_E2E_DATABASE_DSN='postgres://<app-role>:<password>@127.0.0.1:<port>/<database>?sslmode=disable' \
HYDRO_PG_PRIVILEGED_ROLE='<backup-operator-role>' \
npm test -- --project=chromium --workers=1 custom/admin-management.spec.ts
```

The suite also repeats account role/permission/disable persistence checks,
anonymous and rendered privileged-form denials, invitation acceptance,
responsive behavior, and the Admin theme/accessibility matrix.

## Limits

This closes the Editor/Viewer persisted-state slice for IP bans and token
lifetimes. It does not cover denied state changes for every Admin security,
backup, developer, operations, or module-specific mutation. It uses the local
single-Edition fixture; no production identity or security store is certified.

## Token lifetime role × method handler contract — 2026-10-09

The direct registered-handler contract now covers Editor, Viewer, inactive
Admin, and anonymous callers on both GET and valid-CSRF POST. Editor/Viewer
requests carrying an explicit `admin.operations` grant are included to prove a
permission override cannot elevate them to the Admin-only security surface.
Each denied request is checked before settings reads/writes; a prior successful
Admin update remains unchanged. The active Admin success, unsupported duration
validation, and invalid-CSRF denial are also asserted.

```sh
GOCACHE=/tmp/hydro-cover-cache go test ./cmd/hydro \
  -run '^TestAdminTokenLifetimeSettingsApplyAndRejectUnsupportedChoices$' \
  -count=1 -p 1
GOCACHE=/tmp/hydro-cover-cache go vet ./cmd/hydro
```

Both passed. This is handler/middleware evidence and complements the existing
SQLite/PostgreSQL browser persistence evidence; it does not certify production
identity/session deployment.
