# R85 — Explicit tenant scope primitive

Date: 2026-09-19  
Scope: `internal/tenant` and the tenant isolation contract fixture

## Evidence

`internal/tenant` now validates bounded tenant IDs/slugs and cache components,
rejects separators/whitespace/unicode/oversized values, requires an explicit
context scope, and produces deterministic tenant-prefixed cache keys. The
existing isolation contract fixture uses this production package rather than a
test-only context type. Missing scope and cross-tenant key separation are
covered by focused tests.

Command:

```text
go test ./internal/tenant ./tests/contract -run 'Tenant|Scope'
```

Result: passed.

## Scope boundary

This is a reusable foundation only. It does not certify router-to-repository
propagation, PostgreSQL RLS, tenant provisioning/deletion, media/job/audit
isolation, noisy-neighbor capacity, or a shared multi-tenant production mode.
