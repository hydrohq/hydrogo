# R113 — Tenant-scoped form registry

Date: 2026-09-19

## Evidence

- `go test ./internal/forms/sqlstore ./modules/site/... ./cmd/hydro -count=1`
- `go run ./tools/migrationguard`
- `go vet ./...`
- `go test ./... -p 1 -count=1 -timeout=5m`

The focused contract proves a form written by tenant A is not readable or
deletable by tenant B and that missing scope fails closed. PostgreSQL execution,
existing-row backfill and submission routing remain explicit release gates.
