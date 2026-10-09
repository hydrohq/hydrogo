# Admin content permission denied mutations — 2026-10-09

## Scope and result

The real Edition Admin-management browser flow performs page-create,
page-update, and publish POSTs as an authenticated Editor after an Admin
explicitly denies `admin.content.write`. All requests carry a valid session
CSRF token and return 403. A separate authenticated Admin session reads the
page list to verify that the denied create did not add its slug or title, then
opens the existing page and reads it from the Edition API to verify that its
title remains unchanged and its status remains `draft` after the denied update
and publish. The test therefore checks persisted state rather than relying on
error statuses alone.

The same Editor also has `admin.content.read` denied. The import wizard GET
returns 403, and a valid-CSRF import POST without a file also returns 403
rather than reaching file validation. This confirms the write permission check
happens before input processing on that unrendered mutation route. A
valid-CSRF bulk CSV export is also denied with 403 and no download disposition,
so the permission boundary protects this data-export POST as well.

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

This closes only the denied page create/update/publish persistence cells and
the import and bulk-export route read/write denial for an Editor with explicit
content read and write denies. It does not close every Admin route × role ×
permission override × mutation × CSRF combination, module-owned API route, or
production tenant/session deployment gate.
