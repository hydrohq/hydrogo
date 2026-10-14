# R91 — PostgreSQL tenant RLS binding primitive

Date: 2026-09-19

## Evidence

`internal/tenant.BindPostgresTransaction` accepts only a validated scope and a
narrow transaction executor. It binds `app.current_tenant_id` with a
parameterized transaction-local `set_config` call and rejects missing scope or
executor. Focused tests verify the exact SQL boundary and fail-closed behavior.

Command:

```text
go test ./internal/tenant -run TestBindPostgresTransaction -count=1
go vet ./internal/tenant
```

Result: passed.

No RLS table policy or production PostgreSQL isolation claim is made by this
repository-only primitive. Policy migrations, role privileges, call-site
propagation, IDOR/property tests and a real PostgreSQL certification remain
open gates.
