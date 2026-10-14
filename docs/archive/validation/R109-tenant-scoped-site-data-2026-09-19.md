# R109 — Tenant-scoped site data

Date: 2026-09-19

## Evidence

Site settings, navigation, redirects and form inbox storage now use tenant
identity. Composite keys are created for SQLite and PostgreSQL while existing
single-customer data is preserved under the empty namespace. A SQLite contract
test proves scoped settings do not cross tenants and missing scope fails closed.

Commands:

- `go test ./modules/site/... ./cmd/hydro ./migrations -count=1`
- `go run ./tools/migrationguard`

Result: passed.

This is local SQLite evidence. PostgreSQL execution, existing-row backfill,
auth/member/commerce scopes and production restore/export drills remain open.
