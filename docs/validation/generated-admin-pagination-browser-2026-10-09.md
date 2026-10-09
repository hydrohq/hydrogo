# Generated Admin pagination browser check — 2026-10-09

## Scope and result

The native Project Edition generated `project` collection was exercised with
26 records persisted through its API. Chromium verifies that page one shows 25
rows and the forward control, page two shows only the remaining record, and
the backward control returns to page one. This covers the generated Admin's
fixed 25-row page boundary and the browser-visible query-string navigation.

`e2e/custom/generated-admin-crud.spec.ts` passed **1/1** with a fresh SQLite
database (44.6 seconds total) and **1/1** with PostgreSQL 18 (51.4 seconds).
The PostgreSQL application role was `NOSUPERUSER NOBYPASSRLS`; it owned the
disposable database. The same end-to-end case also runs the existing create,
validation, search, filter, edit, delete, theme, and accessibility checks.

Commands:

```sh
# SQLite
npm test -- --project=chromium --workers=1 custom/generated-admin-crud.spec.ts

# PostgreSQL 18 with a fresh database owned by the non-superuser role
HYDRO_E2E_DATABASE_DIALECT=postgres \
HYDRO_E2E_DATABASE_DSN='postgres://<app-role>:<password>@127.0.0.1:<port>/<database>?sslmode=disable' \
npm test -- --project=chromium --workers=1 custom/generated-admin-crud.spec.ts
```

The temporary PostgreSQL server was removed after the run. No production data
or service was used.

## Limits

This verifies forward/back navigation at one page boundary. It does not prove
behavior at the maximum page number, concurrent record changes while paging,
or pagination for every generated collection and theme combination.
