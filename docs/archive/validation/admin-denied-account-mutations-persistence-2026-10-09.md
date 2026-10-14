# Admin account mutation denial persistence — 2026-10-09

## Scope and result

The authenticated Editor/Viewer browser journey now checks the stored target
account after denied role, permission, and disable submissions. Each rejected
request must leave the role and `admin.content.write` override unchanged and
the account active. The check re-reads the account from `/admin/users` through
a separately authenticated Admin session; it does not infer persistence from
the response status alone.

The complete `custom/admin-management.spec.ts` Chromium file passed **5/5** on
SQLite (3.0 minutes) and **5/5** on PostgreSQL 18 (2.8 minutes). PostgreSQL ran
with an application role configured `NOSUPERUSER NOBYPASSRLS` and a separate
`BYPASSRLS` backup operator, on a fresh disposable database. The suite also
covers the existing anonymous and privileged-form role checks, invitation
acceptance, responsive pages, and four-theme light/dark axe matrix.

Commands:

```sh
# SQLite
npm test -- --project=chromium --workers=1 custom/admin-management.spec.ts

# PostgreSQL 18; disposable database and the roles described below
HYDRO_E2E_DATABASE_DIALECT=postgres \
HYDRO_E2E_DATABASE_DSN='postgres://<app-role>:<password>@127.0.0.1:<port>/<database>?sslmode=disable' \
HYDRO_PG_PRIVILEGED_ROLE='<backup-operator-role>' \
npm test -- --project=chromium --workers=1 custom/admin-management.spec.ts
```

The PostgreSQL run used a non-superuser, non-`BYPASSRLS` application role and
a separately provisioned `BYPASSRLS` backup operator. The temporary server was
removed after verification. No production database or credentials were used.

## Limits

This closes the browser persistence assertion for the three account-management
mutations only. It does not prove no side effects for every privileged Admin
POST or module API, nor complete the exhaustive role × route × mutation × CSRF
matrix. The suite's default product database is single-tenant; this is not a
tenant-isolation result.
