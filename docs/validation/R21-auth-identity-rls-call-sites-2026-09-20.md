# R21 — Auth identity RLS call-site binding

## Evidence

- `internal/auth/sqlstore/UserStore` direct CRUD/lookups now use
  `database.NewTenantTransactor`.
- `internal/auth/sqlstore/SessionStore` direct CRUD/lookups now use the same
  transaction boundary.
- `go test ./internal/auth/sqlstore ./internal/auth ./tests/contract` passed.

## Scope limitation

The mobile device session service and member/site/media/transport handlers still
contain direct pool access in some paths. The PostgreSQL isolation drill and a
complete direct-pool inventory are not yet closed, so this is not a shared SaaS
certification.
