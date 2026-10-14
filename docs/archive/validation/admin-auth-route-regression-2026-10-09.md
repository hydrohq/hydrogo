# Admin identity and permission package regression — 2026-10-09

## Scope

Re-run Core Admin authentication, account management, roles, permission
overrides and the associated SQL store package after the recent generated and
module Admin authorization changes.

## Evidence

- `go test ./internal/auth/... -count=1 -p 1 -timeout=6m` passed for
  `internal/auth` and `internal/auth/sqlstore`.
- `go vet ./internal/auth/...` passed.
- The registered Admin account mutations are also mounted on a real
  `http.ServeMux`; all five role/permission/disable/invite/reset-link paths
  return 403 for Editor, Viewer, inactive Admin, and Admin with explicit
  `admin.users.manage=false`. The same routes reject invalid CSRF for an
  active Admin. This route-dispatch test is
  `TestAdminUserMutationRoutesEnforceRolesAndCSRF`.
- After adding that matrix, the focused route test, full `internal/auth/...`
  suite, and `go vet ./internal/auth/...` passed again.
- The workspace has no `HYDRO_TEST_POSTGRES_URL`; PostgreSQL test cases are
  not reported as passing from this run.
- GitHub push was retried separately and failed because this environment
  cannot resolve `github.com`; commit bundles remain available for transfer.

## Limits

This confirms package-level auth/SQL contracts. It does not replace the
repository-wide Admin role × route × mutation matrix, a live PostgreSQL
contract run, or browser testing.
