# Public Admin login confirmation accessibility — 2026-10-09

The login's invite-accepted and password-reset success statuses now have a
browser axe matrix across two locales (English/Turkish) and both supported
color modes (light/dark): eight combinations. These are separate public
confirmation states; no authenticated session or database mutation is needed
to render them.

The first real Chromium/axe run found three WCAG 2.1 AA contrast failures in
light mode:

- The subtle link used `#0284c7` on `#f8fafc` (3.91:1).
- The primary button used white on `#0284c7` (4.09:1).
- The success message used `#10b981` on its pale green background (2.17:1).

The light-mode accent and primary-button background now use `#0369a1`, and
the success-message text uses `#047857`. The focused eight-combination matrix
passed, then the complete six-test Admin-management Chromium file passed
**6/6** on SQLite (3.1 minutes) and **6/6** on PostgreSQL 18.6 (2.7 minutes).
The PostgreSQL browser suite used separate non-superuser app and BYPASSRLS
backup roles; its temporary database container was removed. `go test
./internal/ui -count=1` passed, and the Impeccable mechanical detector returned
no findings for the changed UI and E2E files.

```sh
cd e2e
npm test -- --project=chromium --workers=1 custom/admin-management.spec.ts

HYDRO_E2E_DATABASE_DIALECT=postgres \
HYDRO_E2E_DATABASE_DSN='postgres://<app-role>:<password>@127.0.0.1:<port>/<database>?sslmode=disable' \
HYDRO_PG_PRIVILEGED_ROLE='<backup-operator-role>' \
npm test -- --project=chromium --workers=1 custom/admin-management.spec.ts

cd ..
go test ./internal/ui -count=1
```

The public login has its own light/dark appearance and does not implement the
four Admin Studio shell families. This evidence covers only these two success
states; failed login, MFA challenge/error states, other public authentication
pages, physical devices, and deployed Editions remain separate open checks.
