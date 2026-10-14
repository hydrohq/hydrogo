# R116 — Tenant-scoped auth security tokens and MFA

Date: 2026-09-20

## Evidence

- `go test ./internal/auth/sqlstore ./internal/auth ./cmd/hydro -count=1`
- `go run ./tools/migrationguard`
- `go vet ./...`
- `go test ./... -p 1 -count=1 -timeout=5m`

The focused contract creates invite, recovery and MFA records in tenant A and
proves tenant B cannot consume or read them. Full suite and migration checks
passed; PostgreSQL RLS and production backfill remain explicit release gates.
