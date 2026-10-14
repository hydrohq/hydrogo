# Admin-issued password reset validation

**Date:** 2026-10-07  
**Related decision:** [ADR-0271: Security hardening and recovery controls](../architecture/0271-security-hardening-and-recovery.md)

## Capacity and deployment

- The existing reset endpoint and database contract are reused. No new route,
  worker, service, schema, or migration is introduced; rollback needs no data
  repair.
- Reset form and POST responses are bounded by the existing 16 KiB request
  limit and confirmation limiter. Recovery token consumption, password hashing,
  and session revocation remain in the existing transaction.
- GET and POST responses use `no-store` and `no-referrer`; the page’s exact
  inline token-removal script is added to the strict CSP hash policy.
- Database auth contract validation ran against SQLite and an ephemeral
  PostgreSQL 18.6 container. Existing migration metadata is unchanged.

## Evidence

- `go test ./modules/designsystem ./internal/auth ./internal/ui ./cmd/hydro`
- `go test ./tests/contract -run '^TestPostgresAuthContract$' -count=1`
  with `HYDRO_TEST_POSTGRES_URL` set to the ephemeral PostgreSQL 18.6 instance;
  it covers invite, password recovery, MFA/TOTP and one-time recovery codes.
- Full `go test -p 1 ./... -count=1 -timeout=5m` ran with the PostgreSQL
  contract URL pointed at a fresh database owned by a non-superuser role with
  `NOBYPASSRLS`; the suite exercised PostgreSQL-backed packages as well as
  SQLite tests. The CI contract role was changed to this same least-privilege
  shape so RLS tests cannot accidentally run as the service bootstrap superuser.
- CI's dedicated PostgreSQL contract command,
  `go test ./tests/contract -run '^TestPostgres' -count=1 -p 1`, passed against a
  separate fresh PostgreSQL 18.6 database.
- `go vet ./...` and the CI actionguard, importguard (all three Edition roots),
  moduleguard, migrationguard, and `git diff --check` passed.
- Chromium management E2E creates a reset link from the real admin users UI,
  opens the actual token URL, checks URL cleanup/no-store/no-referrer and axe
  WCAG 2.1 A/AA, verifies weak/mismatched password recovery, changes the
  password, confirms the old password fails and logs in with the new one.
