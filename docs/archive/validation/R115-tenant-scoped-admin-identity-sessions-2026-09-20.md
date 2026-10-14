# R115 — Tenant-scoped admin identity and sessions

Date: 2026-09-20

## Evidence

- `go test ./internal/auth/sqlstore ./internal/auth ./cmd/hydro -count=1`
- `go run ./tools/migrationguard`
- `go vet ./...`
- `go test ./... -p 1 -count=1 -timeout=5m`

The focused contract creates a tenant-A admin and browser session, then proves
tenant B cannot resolve either and a missing scope fails closed. Invite,
recovery, MFA and PostgreSQL RLS remain explicit release gates.
