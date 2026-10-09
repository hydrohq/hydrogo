# Admin content permission denied create persistence — 2026-10-09

## Scope and result

The real Edition Admin-management browser flow performs page-create and
page-update POSTs as an authenticated Editor after an Admin explicitly denies
`admin.content.write`. Both requests carry a valid session CSRF token and
return 403. A separate authenticated Admin session reads the page list to
verify that the denied create did not add its slug or title, then opens the
existing page to verify that its title remains unchanged after the denied
update. The test therefore checks persisted state rather than relying on
error statuses alone.

The focused role-boundary Chromium scenario passed **1/1** on SQLite and
**1/1** on PostgreSQL 18.6. PostgreSQL used an application role configured
`NOSUPERUSER NOBYPASSRLS` and a distinct `BYPASSRLS` backup operator in a
disposable cluster/database; both were removed after the run.

```sh
# SQLite
npx playwright test custom/admin-management.spec.ts --project=chromium \
  --workers=1 --grep 'editor and viewer sessions enforce' --reporter=line

# PostgreSQL 18.6; use a disposable DB and separate app/backup roles
HYDRO_E2E_DATABASE_DIALECT=postgres \
HYDRO_E2E_DATABASE_DSN='postgres://<app-role>:<password>@127.0.0.1:<port>/<database>?sslmode=disable' \
HYDRO_PG_PRIVILEGED_ROLE='<backup-operator-role>' \
npx playwright test custom/admin-management.spec.ts --project=chromium \
  --workers=1 --grep 'editor and viewer sessions enforce' --reporter=line
```

## Limits

This closes only the denied page create/update persistence cells for an Editor
with an explicit content-write deny. It does not close every Admin route × role
× permission override × mutation × CSRF combination, module-owned API route, or
production tenant/session deployment gate.
