# Module Route Validation for Admin Permissions

**Date:** 2026-10-07  
**Status:** Member/site management and page/news content routes validated  
**Scope:** `admin.members.manage`, `admin.site.manage`, `admin.content.read`, and `admin.content.write`.

`GET /admin/members` was audited via the real registered route handler under the following conditions:
editor default, editor explicit allow, viewer explicit allow,
admin explicit deny, and inactive user explicit allow. In the two permitted cases,
the route returned 200; in the remaining three cases, it returned 403.

`POST /admin/site/settings` was audited with an SQLite transactor and recording store:
editor default/deny, viewer allow, admin deny, inactive allow, and admin
default. Permitted POSTs returned 204 and wrote a setting; denied requests
returned 403 and performed no writes.

Page/news mutations were exercised against `admin.content.write`; the content detail API and Admin
list screens were invoked against the `admin.content.read` matrix. The news taxonomy
catalog includes a create/delete form and is therefore within `admin.content.write` scope.
Across combinations of editor, viewer, allow/deny
overrides, and inactive users, permitted requests passed permission
checks, while denied requests received a 403 at handler entry.

`GET /api/pages/{id}`, page preview, `GET /api/news/{id}`, and Admin
list/detail views are protected by `admin.content.read`. In Admin details,
if write permission is absent, pages render with a safe preview iframe, and news renders with a
sanitized rich text view and read-only metadata; no mutation form is present.
API detail handlers previously could return draft JSON under an explicit
read deny condition. Furthermore, read-only Admin list views were unnecessarily requiring write permission.
Tests now verify default reader access, as well as 403 responses for explicit deny and inactive users.

## Commands

```sh
go test ./modules/pages/transport ./modules/news/transport ./modules/members/transport -count=1
go test ./modules/site/transport -run '^TestAdminSiteSettingUsesSiteManageOverrides$' -count=1
go test ./tests/contract -run '^TestCriticalUserJourneysContract$' -count=1
```

Result: relevant route packages and the critical user journeys contract test
passed. Tests cover route-level role defaults, explicit
allow/deny, inactive-user deny, content read/write separation, and site mutation side
effects. Every authorized content route passes handler authorization; the complete
authorization + persistence end-to-end matrix and other permission consumers
remain open in Phase 4.

A Chromium role test was added to verify that a viewer can access the page list,
cannot see import/edit actions, and can view read-only details and the API via `content.read`.
In this restricted sandbox, test setup halted at the `execFileSync("go", ...)` step
with `spawnSync go EPERM`; this browser test does not constitute proof of passing here.
HTTP/component tests for read-only page/news renderers passed in their respective Go
transport packages.

`go test ./... -count=1 -p 1 -timeout=5m` was also attempted. The restricted
network sandbox of this workspace prevented opening loopback ports via `operation not permitted`
in tests using `httptest.NewServer`; consequently, the full Go suite result for the repository
could not be verified here. Re-running the full suite remains open for a CI environment with network/socket permissions.
