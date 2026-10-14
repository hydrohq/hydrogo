# Viewer content permission override — 2026-10-09

## Scope and result

The authenticated Admin grants the Viewer explicit `admin.content.read` and
`admin.content.write` permissions using the Admin permission form. The Viewer
can then open the generated page form and create a draft. A separate Admin
session reads that draft through the Edition API and verifies its persisted
title and draft state. The Admin restores both permissions to `inherit`; the
existing Viewer session is revoked (`/auth/me` returns 401).

The focused `editor and viewer sessions enforce the Edition role boundary`
Chromium scenario passed **1/1** on SQLite and **1/1** on PostgreSQL 18.6. The
PostgreSQL run used a disposable cluster, a `NOSUPERUSER NOBYPASSRLS`
application role, and a separate backup operator. This verifies that an
explicit permission grant changes the actual Editor Studio flow and is
reversible without leaving the Viewer session active.

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

This closes one live Viewer allow-override flow for page creation and session
revocation. It does not establish every capability override across every Admin
route or module API, nor production Edition identity/session behavior.
