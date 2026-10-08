# Admin Studio management browser verification on PostgreSQL

Date: 2026-10-08

## Scope and result

The complete `custom/admin-management.spec.ts` Chromium suite ran against a
fresh HydroGo database using the repository migrations and a real PostgreSQL
18 server. All five browser scenarios passed. The same five scenarios passed
against the temporary SQLite fixture.

The suite was rerun after stabilizing its login-session use: current SQLite
and PostgreSQL 18 runs both passed **5/5**. The PostgreSQL rerun also repeated
the administrator backup-download path with the `hydro_backup` BYPASSRLS role.
A setup retry exposed that per-database default privileges must be installed
inside `hydrogo_e2e` before applying migrations; after correcting that fixture
scope, the signed backup download and complete browser suite passed.

The SQLite suite was rerun on 2026-10-08 after the permission and route-matrix
review: `npx playwright test custom/admin-management.spec.ts --project=chromium
--reporter=line --output=/tmp/hydro-admin-management-artifacts` completed with
5/5 passing in 2.6 minutes. Playwright artifacts were directed to `/tmp`
because the checkout's ignored `e2e/artifacts` directory was owned by root.

The scenarios cover the Edition editor/viewer role boundary, anonymous access
and administrator management routes, responsive layouts, invitation
acceptance followed by editor login, and WCAG 2.1 A/AA checks across the four
Studio themes in light and dark modes. The authenticated admin journey also
downloads a backup, so this run exercised the PostgreSQL backup path.

Commands:

```sh
# PostgreSQL; database must be empty and the dump operator configured below.
HYDRO_E2E_DATABASE_DIALECT=postgres \
HYDRO_E2E_DATABASE_DSN='postgres://<app-role>:<password>@127.0.0.1:<port>/<database>?sslmode=disable' \
HYDRO_PG_PRIVILEGED_ROLE='<backup-operator-role>' \
npm test -- --project=chromium --workers=1 custom/admin-management.spec.ts

# SQLite default fixture
npm test -- --project=chromium --workers=1 custom/admin-management.spec.ts
```

For PostgreSQL, the backup operator must be a separately provisioned role with
`BYPASSRLS`, membership granted to the application connection role, and read
access to the database objects. Grant default table and sequence read
privileges for the migration owner before migrations run. This is the operator
boundary in [ADR-0207](../architecture/0207-postgres-backup-role-boundary.md);
HydroGo does not elevate the application role or bypass the boundary itself.

## Findings and test fixture changes

The first PostgreSQL run returned HTTP 500 for the Admin backup download because
the test connection did not configure the required backup operator role. The
PostgreSQL log confirmed `pg_dump` rejected the forced row-level-security table.
After configuring the documented operator role, the same browser journey
downloaded the backup successfully.

The end-to-end fixture now accepts the selected database dialect and DSN in its
role and invitation seed helpers. PostgreSQL runs fail early when the DSN or
backup operator role is missing. The invitation test also honors the login
endpoint's `Retry-After` response once: earlier tests share one loopback client
IP and can consume the per-minute authentication allowance before the invited
editor signs in.

The browser checks test the normal Edition application role and the explicit
backup role boundary. They do not provision production credentials, modify
production database grants, or add a PostgreSQL migration.

## Anonymous management-route matrix rerun — 2026-10-08

The SQLite Chromium suite was rerun after extending its anonymous-access check
from `/admin/users` to 16 management entry points: dashboard, page/news create,
media, users, permissions, security and token lifetimes, audit, backup, control,
operations, developer and resource telemetry, reports, and report export. Each
route returned an authentication response (401 or a redirect to `/login`). The
complete `custom/admin-management.spec.ts` file passed **5/5** in 3.3 minutes,
including its editor/viewer role checks, CSRF checks, responsive flows, invite
acceptance, and theme accessibility matrix. This added anonymous check ran on
SQLite; it does not replace the already recorded PostgreSQL suite or cover all
Admin POST mutation/permission combinations.
