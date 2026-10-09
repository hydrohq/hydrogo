# Admin content permission denied create persistence — 2026-10-09

## Scope and result

The real Edition Admin-management browser flow now performs a page-create POST
as an authenticated Editor after an Admin explicitly denies
`admin.content.write`. The request carries a valid session CSRF token and
returns 403. A separate authenticated Admin session then reads the page list
and verifies that neither the denied slug nor title exists. The test therefore
checks persisted state rather than relying on the error status alone.

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

This closes only the denied page-create persistence cell for an Editor with an
explicit content-write deny. It does not close every Admin route × role ×
permission override × mutation × CSRF combination, module-owned API route, or
production tenant/session deployment gate.
