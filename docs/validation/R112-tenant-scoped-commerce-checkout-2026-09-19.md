# R112 — Tenant-scoped commerce checkout and payments

Date: 2026-09-19

## Scope

Validated the SQLite checkout boundary for orders, order lines, carts, payment
transactions, webhook receipts and refund/expiry paths. The migration adds
tenant discriminators and indexes; the shared-mode store requires the resolved
tenant context for reads and mutations.

## Evidence

- `go test ./modules/commerce/repository/sqlstore ./cmd/hydro -count=1`
- `go run ./tools/migrationguard`
- `go vet ./...`
- `go test ./... -p 1 -count=1 -timeout=5m`

The focused contract proves tenant A's order and cart state is not visible to
tenant B and that a missing order scope fails closed. PostgreSQL execution,
RLS, webhook host routing and production backfill remain explicit release gates.
