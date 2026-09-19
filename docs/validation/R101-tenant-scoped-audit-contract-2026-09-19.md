# R101 — Tenant-scoped audit contract

Date: 2026-09-19

## Evidence

The audit migration adds tenant identity and an indexed lookup path for both
dialects. The store routes validated tenant contexts to scoped writes and
rejects missing scope in explicit fail-closed mode. SQLite tests prove tenant
identity and request correlation are persisted.

Commands:

- `go test ./internal/audit/sqlstore ./internal/database ./internal/tenant -count=1`
- `go vet ./internal/audit/sqlstore ./internal/database ./internal/tenant`

Result: passed.

This is a write-boundary foundation, not tenant audit isolation certification.
Tenant-aware list/export/erasure workflows, PostgreSQL execution, RLS and
operator authorization tests remain open.
