# R106 — Tenant-scoped worker composition

Date: 2026-09-19

## Evidence

The shared-tenancy composition root now enables fail-closed scope requirements
for audit, outbox and media stores. The webhook worker executes each polling
pass under the validated tenant scopes from the host map; tenancy-disabled
deployments preserve the legacy single-tenant path.

Commands:

- `go test ./internal/webhooks ./cmd/hydro -count=1`
- `go vet ./internal/webhooks ./cmd/hydro`

Result: passed.

This is local composition evidence. PostgreSQL RLS policy certification,
dynamic tenant provisioning and multi-instance worker drills remain open.
