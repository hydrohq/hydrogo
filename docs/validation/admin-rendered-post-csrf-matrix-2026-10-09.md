# Rendered Admin POST CSRF matrix — 2026-10-09

The Admin-management browser test discovers same-origin POST form and button
`formaction` targets on ten privileged pages: users, permissions, security,
token lifetimes, audit, backup, control, operations, developer, and developer
resources. For each discovered target, both Editor and Viewer sessions submit
three requests: valid CSRF, missing CSRF, and a deliberately mismatched CSRF
token. Every request must return HTTP 403. The authenticated Admin also
submits missing and mismatched CSRF requests to each target and must receive
403, proving the CSRF boundary independently of Editor/Viewer role denial.
The Admin test first verifies that its CSRF cookie matches the server-issued
token.

After adding the Admin invalid-token cases, the complete
`custom/admin-management.spec.ts` Chromium file passed **5/5** on SQLite
(3.1 minutes) and **5/5** on PostgreSQL 18.6 (2.8 minutes). PostgreSQL used a
`NOSUPERUSER NOBYPASSRLS` app role and a separate `BYPASSRLS` backup operator.
The temporary database container was removed after the run.

```sh
# SQLite
npm test -- --project=chromium --workers=1 custom/admin-management.spec.ts

# PostgreSQL 18.6
HYDRO_E2E_DATABASE_DIALECT=postgres \
HYDRO_E2E_DATABASE_DSN='postgres://<app-role>:<password>@127.0.0.1:<port>/<database>?sslmode=disable' \
HYDRO_PG_PRIVILEGED_ROLE='<backup-operator-role>' \
npm test -- --project=chromium --workers=1 custom/admin-management.spec.ts
```

This is rendered-form evidence. It does not enumerate API-only or
module-owned routes, attempt every valid-CSRF Admin mutation, cover Admin
permission-deny overrides for every action, or prove persisted-state
invariance for each discovered target. Those remain separate acceptance
cells in the audit plan.
