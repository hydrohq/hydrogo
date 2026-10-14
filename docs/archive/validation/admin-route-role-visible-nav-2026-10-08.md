# Admin route role and navigation smoke — 2026-10-08

## Coverage added

`e2e/custom/admin-management.spec.ts` now reads the enabled Edition Admin navigation from an authenticated Admin session and opens every unique `/admin` navigation path, requiring HTTP 200. It then signs in as editor and viewer, reads each role's visible navigation, and verifies every visible path also returns HTTP 200. Existing explicit assertions continue to require HTTP 403 from protected account-management, security, audit, backup, control, operations, and developer routes; they also exercise CSRF-less writes and valid-CSRF editor/viewer attempts to change user roles, permissions, accounts, and page content.

The editor/viewer role test also opens `/admin/security/token-lifetimes` and
`/admin/reports/export?range=week` directly, and submits the token-lifetime
mutation with a valid CSRF token. Both administrator-only routes return HTTP 403
for both roles, including when the POST carries valid CSRF.

The test intentionally does not infer denial from a hidden navigation link: the viewer can read some content routes that are omitted from its menu. Authorization is checked from the HTTP response on explicitly protected paths.

## Command palette permission coverage

The same role test now checks the quick command palette against each role's
effective capabilities. Editors see page/news creation and media upload, but
not commerce, member, reporting, backup, security, control, or operations
commands. Viewers see none of those mutations or privileged commands. When an
administrator denies `admin.content.write`, page/news commands disappear from
the editor's palette along with the denied routes; after an explicit allow and
fresh login, both commands return. The independently authorized media command
remains available throughout the content permission change.

The focused editor/viewer browser journey passes on SQLite and PostgreSQL;
full route-by-route permission and mutation permutations remain open.

## Full Admin management browser rerun

The complete five-test Chromium file passes on SQLite. The rerun caught two
test-only issues in the longer admin management journey: after content-read is
denied, `/admin/pages` correctly returns a JSON 403 and therefore has no Studio
command palette; and repeated administrator/editor sign-ins exceeded the real
per-IP limit of five logins per minute. The test now checks palette filtering
through the search API for the denied role, reuses the authenticated admin
session captured in the preceding role journey, and opens a fresh editor
session after changing permissions.

```sh
cd e2e
TMPDIR=/home/arda/hydrogo/.tmp/pw-system-tmp npx playwright test \
  custom/admin-management.spec.ts --project=chromium --workers=1 \
  --reporter=line --trace=off \
  --output=/home/arda/hydrogo/.tmp/admin-management-full-final2
```

Result: 5/5 passed in 2.2 minutes. The login limit itself was not changed.

Focused command palette run:

```sh
cd e2e
npx playwright test custom/admin-management.spec.ts --project=chromium \
  --workers=1 --grep 'editor and viewer sessions enforce' --reporter=line

HYDRO_E2E_DATABASE_DIALECT=postgres \
HYDRO_E2E_DATABASE_DSN='postgres://<test-role>:<password>@127.0.0.1:<port>/<database>?sslmode=disable' \
HYDRO_PG_PRIVILEGED_ROLE=unused_for_this_focused_test \
npx playwright test custom/admin-management.spec.ts --project=chromium \
  --workers=1 --grep 'editor and viewer sessions enforce' --reporter=line
```

The focused PostgreSQL test does not download a backup; the operator role value
only satisfies the shared fixture's configuration check. The disposable
PostgreSQL 18 database was removed after the run.

## Full Admin management suite rerun — 2026-10-09

The complete five-test Chromium file passed 5/5 in 1.9 minutes, including the
authenticated Editor/Viewer role journey, all listed Admin management routes,
permission-denied account behavior, mobile/tablet layout, invitation link
creation/error feedback, and WCAG 2.1 A/AA scans across four Studio themes in
both color modes.

The browser file now restarts its server process between passing serial test
journeys while retaining the same database. This isolates each journey from the
real process-local limit of five login attempts per IP per minute; it does not
change or bypass that production limit within any individual journey. SQLite
users, permissions, and DB-backed sessions persist across each restart.

```sh
cd e2e
npx playwright test --project=chromium --workers=1 custom/admin-management.spec.ts \
  --reporter=line --trace=off
```

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
