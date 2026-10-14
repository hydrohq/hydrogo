# Media cleanup permission override — 2026-10-09

## Finding and change

Media cleanup required the Admin role but bypassed explicit user overrides
for `admin.media.manage`, unlike the rest of the media Admin surface. Cleanup
plan and deletion now require both the Admin role and
`auth.Can(user, auth.PermissionMediaManage)`. The action remains Admin-only;
an Editor granted media management cannot run garbage collection.

## Verification

- The route-derived denial test covers media list, upload, metadata, folders,
  settings, visibility, cleanup plan, and cleanup deletion for an Editor
  denied media access, a default Viewer, and an Admin explicitly denied media
  access. Unsafe mutations with invalid CSRF are denied before handlers.
- SQLite-backed cleanup route tests prove an Admin denied `admin.media.manage`
  receives 403 and no asset is deleted; the normal Admin flow still deletes
  the selected unreferenced asset only after confirmation.
- Focused authorization/CSRF/cleanup tests passed:
  `go test ./internal/media -run 'TestDeniedMediaPermissionStopsAdminHandlersBeforeDependencies|TestAdminMediaCleanupRequiresAdminAndConfirmation|TestAdminMediaCleanupRouteRequiresCSRFBeforeDeleting' -count=1 -p 1 -timeout=4m`.
- Full media module tests passed:
  `go test ./internal/media/... -count=1 -p 1 -timeout=5m`.
- `go vet ./internal/media` and `git diff --check` passed.

## Limits

This is local SQLite/in-process evidence. PostgreSQL, browser workflows, and
customer deployment role provisioning remain open gates.
