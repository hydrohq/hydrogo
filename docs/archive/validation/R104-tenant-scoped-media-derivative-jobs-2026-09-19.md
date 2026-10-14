# R104 — Tenant-scoped media derivative jobs

Date: 2026-09-19

## Evidence

The optimizer restores scope from a namespaced asset key before derivative
repository calls. Ready variant keys are namespaced by the scoped SQL store.
Tests cover key-to-scope restoration and prevent cross-tenant variant reads;
legacy derivative tests remain green.

Commands:

- `go test ./internal/media/... ./internal/database ./internal/tenant -count=1`
- `go vet ./internal/media/... ./internal/database ./internal/tenant`

Result: passed.

This is a local queued-job contract, not production worker isolation evidence.
Provider-backed derivative storage, outbox worker propagation and load/property
tests remain open.
