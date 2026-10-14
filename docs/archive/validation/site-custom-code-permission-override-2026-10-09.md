# Site custom code permission override — 2026-10-09

## Finding and change

The custom-code route required an active Admin role but ignored the
`admin.site.manage` permission override. That differed from the shared Site
Admin permission contract and allowed an explicitly denied Admin to access
the route. The handler now requires both the Admin role and
`auth.Can(user, auth.PermissionSiteManage)`.

## Verification

- A migrated SQLite HTTP test rejects an Admin with explicit
  `admin.site.manage=false` for custom-code mutation and proves the private
  setting remains absent.
- The route-derived GET matrix includes an Admin with the same explicit deny
  and verifies every Site Admin deep link returns HTTP 403.
- The authorized Admin path still persists and reads back the private setting;
  existing CSRF and editor restrictions remain covered.
- `go test ./modules/site/transport -run 'TestSite(AdminDeepLinksDenyEditorAndViewer|CustomCodeAdminRouteIsPrivateAndAdminOnly|AdminMutationRoutesEnforceCSRFAndSiteManage)$' -count=1 -p 1 -timeout=4m` passed.
- `go test ./modules/site/... -count=1 -p 1 -timeout=5m` passed.
- `go vet ./modules/site/transport` and `git diff --check` passed.

## Limits

This is SQLite and in-process HTTP evidence. PostgreSQL, browser interaction,
and deployment-level role provisioning remain open gates.
