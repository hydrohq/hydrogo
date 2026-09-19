# R111 — Tenant-scoped commerce catalog

Date: 2026-09-19

## Scope

Validated the SQLite product/variant repository boundary for shared mode. The
catalog migration adds tenant discriminators and indexes; create, list, get and
stock-reservation paths use the validated request scope.

## Evidence

- `go test ./modules/commerce/repository/sqlstore ./cmd/hydro -count=1`
- `go run ./tools/migrationguard`
- `go vet ./...`
- `go test ./... -p 1 -count=1 -timeout=5m`

The focused test proves tenant A cannot read tenant B's catalog and that a
missing scope fails closed. PostgreSQL execution and production RLS remain open
release gates, as do order/cart/payment tenant predicates.
