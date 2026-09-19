# R103 — Tenant object-storage namespace

Date: 2026-09-19

## Evidence

Tenant-scoped uploads now derive `tenants/{tenant_id}/{basename}` keys before
metadata persistence and BlobStore writes. Legacy contexts retain the existing
basename behavior. Tests prove deterministic key derivation and that the actual
BlobStore receives the namespaced key.

Commands:

- `go test ./internal/media ./internal/media/sqlstore -count=1`
- `go vet ./internal/media/... ./internal/database ./internal/tenant`

Result: passed.

This is an upload/key-boundary contract, not full object isolation certification.
Public tenant routing, derivative jobs, migration/backfill and provider-level
signed URL drills remain open.
