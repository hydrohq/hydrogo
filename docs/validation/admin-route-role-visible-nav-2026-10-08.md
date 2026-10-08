# Admin route role and navigation smoke — 2026-10-08

## Coverage added

`e2e/custom/admin-management.spec.ts` now reads the enabled Edition Admin navigation from an authenticated Admin session and opens every unique `/admin` navigation path, requiring HTTP 200. It then signs in as editor and viewer, reads each role's visible navigation, and verifies every visible path also returns HTTP 200. Existing explicit assertions continue to require HTTP 403 from protected account-management, security, audit, backup, control, operations, and developer routes; they also exercise CSRF-less writes and valid-CSRF editor/viewer attempts to change user roles, permissions, accounts, and page content.

The editor/viewer role test also opens `/admin/security/token-lifetimes` and
`/admin/reports/export?range=week` directly, and submits the token-lifetime
mutation with a valid CSRF token. Both administrator-only routes return HTTP 403
for both roles, including when the POST carries valid CSRF.

The test intentionally does not infer denial from a hidden navigation link: the viewer can read some content routes that are omitted from its menu. Authorization is checked from the HTTP response on explicitly protected paths.

## Result

Commands:

```sh
cd e2e && npx playwright test custom/admin-management.spec.ts --project chromium --workers=1 --reporter=line --output=/tmp/hydro-admin-route-matrix
HYDRO_E2E_DATABASE_DIALECT=postgres \
HYDRO_E2E_DATABASE_DSN='postgres://<app-role>:<password>@127.0.0.1:<port>/<database>?sslmode=disable' \
HYDRO_PG_PRIVILEGED_ROLE='<backup-operator-role>' \
npx playwright test custom/admin-management.spec.ts --project chromium --workers=1 \
  --grep 'editor and viewer sessions' --reporter=line
```

Result: full SQLite Chromium suite 5/5 passed before the added deep-link assertions; focused editor/viewer Chromium test passed on SQLite and PostgreSQL after the assertions were added. The test server was stopped after each run.

## Remaining scope

The test covers currently active navigation paths for this Edition fixture and selected administrator-only deep links. It does not cover every module-specific route or every mutation form under each role and permission override. Full route and mutation permutation coverage remains a separate expansion.
