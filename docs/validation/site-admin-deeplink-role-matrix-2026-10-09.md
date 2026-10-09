# Site Admin deep-link role matrix — 2026-10-09

## Scope

Exercise direct HTTP GET access to every non-public `/admin/...` route
registered by `modules/site/transport.Handler.Routes()` with Forms and Fonts
enabled. The test derives its cases from the runtime route table, so newly
registered Site Admin deep links join the matrix automatically.

## Evidence

- Every registered management GET deep link returned HTTP 403 for an active
  Editor and an active Viewer, including route parameters and optional module
  routes.
- Existing `TestSiteAdminMutationRoutesEnforceCSRFAndSiteManage` derives the
  Site Admin POST cases from the same route table and checks invalid CSRF,
  Editor and Viewer roles, explicit Admin `site.manage=false`, and inactive
  accounts.
- `go test ./modules/site/transport -run '^TestSiteAdminDeepLinksDenyEditorAndViewer$' -count=1 -p 1 -timeout=3m` passed.
- `go test ./modules/site/transport -count=1 -p 1 -timeout=5m` passed.
- `go vet ./modules/site/transport` and `git diff --check` passed.

## Limits

This proves Site module route behavior in-process. It does not cover every
Admin module, rendered menu visibility, browser journeys, PostgreSQL, or a
deployed customer Edition. Those remain separate acceptance gates.
