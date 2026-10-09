# Admin identity and permission package regression — 2026-10-09

## Scope

Re-run Core Admin authentication, account management, roles, permission
overrides and the associated SQL store package after the recent generated and
module Admin authorization changes.

## Evidence

- `go test ./internal/auth/... -count=1 -p 1 -timeout=6m` passed for
  `internal/auth` and `internal/auth/sqlstore`.
- `go vet ./internal/auth/...` passed.
- The workspace has no `HYDRO_TEST_POSTGRES_URL`; PostgreSQL test cases are
  not reported as passing from this run.
- GitHub push was retried separately and failed because this environment
  cannot resolve `github.com`; commit bundles remain available for transfer.

## Limits

This confirms package-level auth/SQL contracts. It does not replace the
repository-wide Admin role × route × mutation matrix, a live PostgreSQL
contract run, or browser testing.
