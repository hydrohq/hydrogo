# R100 — Tenant-context outbox propagation

Date: 2026-09-19

## Evidence

The outbox SQL store now routes valid context scopes to the tenant-filtered
enqueue/claim/ack/retry operations. In explicit `RequireTenantScope` mode it
rejects missing scope. Tests prove automatic routing, fail-closed behavior, and
the existing legacy lease contract.

Commands:

- `go test ./internal/outbox/sqlstore ./internal/outbox ./internal/tenant -count=1`
- `go vet ./internal/outbox/sqlstore ./internal/outbox ./internal/tenant`

Result: passed.

This is a context/storage bridge, not shared-tenancy certification. Router scope
resolution, composition-root configuration, all repository call-site review,
PostgreSQL RLS and production isolation tests remain open.
