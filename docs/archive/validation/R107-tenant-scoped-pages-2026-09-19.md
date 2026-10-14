# R107 — Tenant-scoped page content

Date: 2026-09-19

## Evidence

The pages migration adds tenant discriminators to content and revisions, and
the repository scopes published/admin lists, route reads, draft writes, updates
and publishes. A SQLite contract test proves tenant A cannot list tenant B and
that missing scope fails closed.

Commands:

- `go test ./modules/pages/repository/sqlstore ./modules/pages/... ./cmd/hydro -count=1`
- `go test ./migrations -count=1`
- `go run ./tools/migrationguard`

Result: passed.

Existing content backfill and the remaining content/member repository scopes are
not certified by this local evidence.
