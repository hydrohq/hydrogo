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

## Editor/viewer media mutation boundary — 2026-10-08

The role journey now exercises an actual persisted media mutation. An editor
with the default `admin.media.manage` capability creates a folder with a valid
CSRF token; the redirect succeeds and the folder appears in the editor's media
listing and a later admin listing. A viewer submits the same operation with a
valid CSRF token and receives 403; the attempted folder is absent from the
admin listing. The complete management Chromium file passed 5/5 on SQLite
(3.1 minutes) and PostgreSQL 18 (4.5 minutes) using fresh disposable test
databases. Targeted Go tests for `internal/media`, `internal/auth`, and
`cmd/hydro` passed. The PostgreSQL container and database were removed after
the run.

This proves the folder-create capability boundary and persistence on both
adapters. The contract route checks below now cover successful SQLite
persistence for upload, metadata, visibility, and settings; their PostgreSQL
result awaits a configured contract database. Cleanup's handler test uses test
doubles, so adapter persistence remains open.

## Media route, CSRF, and role deny matrix — 2026-10-08

`internal/media` now mounts the real Admin media route handlers behind the
production CSRF middleware in an in-memory `http.ServeMux` and tests the
editor-with-explicit-deny and default viewer cases for list, upload, metadata,
folder creation, settings, visibility, cleanup planning, and cleanup mutation.
Every POST rejects a wrong CSRF token in middleware, then rejects the same
request with a valid token at the role boundary before touching storage
dependencies. The GET list and cleanup plan are denied at the role boundary.
The focused Go test passed:

```sh
go test ./internal/media -run 'TestMediaAccessUsesPermissionOverrides|TestDeniedMediaPermissionStopsAdminHandlersBeforeDependencies' -count=1
```

An additional in-memory route test now submits an invalid token and verifies
that cleanup does not run, then submits a valid token as an active Admin and
verifies the cleanup repository receives the deletion. This uses SQLite for
the collector transaction but test doubles for media metadata and blob
storage; it is not an adapter persistence or browser result. The expanded
focused command is:

```sh
go test ./internal/media -run 'TestMediaAccessUsesPermissionOverrides|TestDeniedMediaPermissionStopsAdminHandlersBeforeDependencies|TestAdminMediaCleanupRouteRequiresCSRFBeforeDeleting|TestAdminMediaCleanupRequiresAdminAndConfirmation' -count=1
```

The settings mutation test now also traverses `AdminRoutes()` and the CSRF
middleware as an active editor: an invalid token is rejected without saving,
and a valid token saves settings. Its repository and cache are test doubles,
so it validates the HTTP/authorization flow rather than durable settings
storage. The combined focused command is:

```sh
go test ./internal/media -run 'TestMediaAccessUsesPermissionOverrides|TestDeniedMediaPermissionStopsAdminHandlersBeforeDependencies|TestAdminMediaCleanupRouteRequiresCSRFBeforeDeleting|TestAdminMediaCleanupRequiresAdminAndConfirmation|TestAdminMediaSettingsPersistAndApplyHotlinkProtection' -count=1
```

`tests/contract` now tests the metadata mutation through the real Admin route
and CSRF middleware against SQLite and PostgreSQL media stores. An invalid
token leaves the persisted asset unchanged; an active editor with a valid
token updates focal coordinates, folder, tags, and visibility, which are read
back from the database. The same route test covers multipart upload: invalid
CSRF creates no asset row, while a valid editor request writes the file to the
configured local media directory and persists a ready/private asset assigned
to its selected folder. It also saves media settings through the route and
checks their persisted SQLite/PostgreSQL row is visible only in the tenant
resolved from the host. The SQLite case passed locally. The PostgreSQL case was
registered but skipped because `HYDRO_TEST_POSTGRES_URL` is not configured in
this environment; it is not counted as a PostgreSQL result:

```sh
go test ./tests/contract -run 'Test(SQLite|Postgres)AdminMediaMutationRoutesCSRFAndPersistence' -count=1
```

PostgreSQL execution and full browser coverage for these mutations remain
open. The SQLite cleanup adapter gap was closed by the later contract section
below; PostgreSQL and browser coverage remain open.
The route test also uses the production host resolver with two tenant host
mappings. It proves that a forged `X-Tenant-ID` does not change the resolved
scope, a mutation through the other tenant's host cannot change the asset,
and the asset remains invisible when read with the other tenant's scope.

These route and contract checks run without opening a network listener; the
contract test proves successful persisted state on SQLite as described above.
Rendered browser flows remain unverified here. Earlier SQLite/PostgreSQL
browser evidence above remains valid for folder creation only; upload,
metadata, visibility, settings, and cleanup still need browser coverage when
the fixture can start.

A fresh Playwright attempt for the editor/viewer management scenario used a
`/tmp` output directory to avoid the repository's read-only Playwright artifact
directory. The fixture failed before migration or browser startup because
Node could not start its `go run` child process (`spawnSync go EPERM`). This is
not a browser pass; the in-memory and database contract tests above are the
available evidence for this turn.

The full `go test ./internal/media -count=1` package run was also attempted.
It stopped in the existing `TestPrivateMediaLiveHTTPRejectsAnonymous` test
because `httptest.NewServer` could not bind a loopback port (`socket:
operation not permitted`). That test now exercises the same auth/handler
behavior through an in-memory mux. The full `go test ./internal/media/... -count=1`
package run passes; this does not verify a real network listener.

## Persistent media cleanup route contract — 2026-10-08

The GC contract now exercises the production cleanup GET/POST handlers through
the CSRF middleware with a real SQL media repository, tenant-scoped reference
scanner, and SQLite/PostgreSQL adapter path. It verifies that the preview sums
the master asset and thumbnail sizes, a content reference hides the asset from
the plan, an invalid CSRF token leaves database rows and storage keys intact,
and a valid Admin request deletes both object keys plus the media and variant
rows. A repeated cleanup request deletes nothing.

`go test ./tests/contract -run 'Test(SQLite|Postgres)MediaGarbageCollectionContract' -count=1`
passed for SQLite. The PostgreSQL case is registered but skipped locally
because `HYDRO_TEST_POSTGRES_URL` is unset; this is not PostgreSQL evidence.
The contract uses a blob-store test double, so provider-specific object deletion
and a browser journey remain unverified.

### Retry after partial object-store failure

The SQLite contract injects a one-time failure while deleting the thumbnail,
after the master object was deleted. The Admin route returns 503 and preserves
the media row in `deleting` state. A second confirmed request retries both
object keys and removes the media and variant rows; repeating cleanup after
success deletes nothing. The contract test and `go vet ./tests/contract` pass.
The PostgreSQL case was not run, and provider-specific idempotency remains
unverified because the contract uses an object-store test double.

## PostgreSQL environment recheck — 2026-10-09

`HYDRO_TEST_POSTGRES_URL` is unset in the current execution environment. A
temporary PostgreSQL 18.6 cluster initialized successfully under `/tmp`, but
`pg_ctl` could not create its Unix-domain socket (`Operation not permitted`);
the server did not start, so no PostgreSQL contract result is claimed from this
attempt. TCP loopback listeners are also unavailable in this sandbox. Existing
PostgreSQL results recorded above remain limited to their named runs and do not
close the pending tests. To run the remaining contracts, configure
`HYDRO_TEST_POSTGRES_URL` in a PostgreSQL-capable environment and run the
documented `go test ./tests/contract` and `go test ./...` commands there.
