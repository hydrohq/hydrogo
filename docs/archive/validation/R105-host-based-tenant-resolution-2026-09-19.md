# R105 — Host-based tenant resolution

Date: 2026-09-19

## Evidence

The runtime accepts an operator-owned `tenancy.hosts` map, establishes a
validated context scope from the normalized host, ignores spoofed tenant
headers, and fails closed for unknown hosts when enabled. Health/readiness are
kept available for load balancers. Config and server integration tests pass.

Commands:

- `go test ./internal/tenant ./internal/config ./internal/server ./cmd/hydro -count=1`
- `go vet ./internal/tenant ./internal/config ./internal/server ./cmd/hydro`

Result: passed.

This is local boundary evidence. DNS/TLS/custom-domain proxy preservation,
provisioning and multi-tenant production routing drills remain open.
