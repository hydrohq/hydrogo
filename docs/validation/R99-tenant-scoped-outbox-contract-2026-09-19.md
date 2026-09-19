# R99 — Tenant-scoped outbox contract

Date: 2026-09-19

## Evidence

Migration `202609190003_outbox_tenant_scope` adds the tenant discriminator and
claim index for both dialects. The SQL store's scoped enqueue/claim/ack/retry
operations require an explicit validated tenant context. The SQLite contract
proves a tenant cannot claim or acknowledge another tenant's event, while the
legacy lease contract remains green.

Commands:

- `go test ./internal/outbox/... ./internal/database ./internal/tenant -count=1`
- `go vet ./internal/outbox/... ./internal/database ./internal/tenant`

Result: passed.

This is a storage-boundary foundation, not shared-tenancy certification. Scoped
call-site propagation, worker selection, PostgreSQL execution, RLS policies,
IDOR/property tests and noisy-neighbor capacity evidence remain open.
