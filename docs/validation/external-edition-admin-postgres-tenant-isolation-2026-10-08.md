# Independent Edition PostgreSQL tenant Admin browser verification

Date: 2026-10-08

The independent Edition Chromium test server now accepts `--dialect postgres`
and connects with pgx. The browser fixture still defaults to SQLite; set
`HYDRO_E2E_DATABASE_DIALECT` and `HYDRO_E2E_DATABASE_DSN` to use a disposable,
empty PostgreSQL database.

On PostgreSQL 18 with a `NOSUPERUSER NOBYPASSRLS` database role, the real
generated migration and Admin/API routes passed the cross-tenant browser flow
(1/1). Tenant A created a record; tenant B could not see it in the generated
Admin list or retrieve it by API (404). Tenant A edited and deleted it. A
viewer was denied the Admin route (403), and a request without the valid test
CSRF token was denied (403) without creating an API record.

This validates generated repository tenant predicates and the Admin/API
composition against PostgreSQL. Tenant identity and role come from explicit
test-only headers; this run does not certify production identity resolution or
PostgreSQL row-level-security policy wiring.

```sh
HYDRO_E2E_DATABASE_DIALECT=postgres \
HYDRO_E2E_DATABASE_DSN='postgres://<test-role>:<password>@127.0.0.1:<port>/<empty-test-database>?sslmode=disable' \
npm test -- --project=chromium --workers=1 custom/external-edition-admin.spec.ts

# SQLite remains the default.
npm test -- --project=chromium --workers=1 custom/external-edition-admin.spec.ts
```
