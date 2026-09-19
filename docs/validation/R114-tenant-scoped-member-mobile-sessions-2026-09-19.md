# R114 — Tenant-scoped member mobile sessions

Date: 2026-09-19

## Evidence

- `go test ./modules/members/... ./cmd/hydro -count=1`
- `go run ./tools/migrationguard`
- `go vet ./...`
- `go test ./... -p 1 -count=1 -timeout=5m`

The focused contract issues a token in tenant A and proves access validation,
missing-scope validation and device listing cannot use it from tenant B. The
member session migration and indexes were already certified in R110. Admin
auth identity stores and PostgreSQL RLS remain release gates.
