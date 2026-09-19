# R90 — Tenant-scoped cache API

Date: 2026-09-19

## Evidence

`internal/cache.Cache` and `internal/cache.TwoTier` now provide
`GetOrLoadScoped`. The method requires `internal/tenant` scope, prefixes both
local and remote keys with the validated tenant ID, and refuses missing scope
or malformed components. Tests prove two tenants can use the same logical
`pages/home` key without a cross-tenant hit, and that an unscoped request fails
closed.

Command:

```text
go test ./internal/cache ./internal/tenant -count=1
go vet ./internal/cache ./internal/tenant
```

Result: passed.

This is an opt-in cache boundary, not proof that every existing application
call site is scoped. Repository/media/job/audit propagation and PostgreSQL RLS
remain open multi-tenant gates.
