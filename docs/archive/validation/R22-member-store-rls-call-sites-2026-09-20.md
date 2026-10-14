# R22 — Member store RLS call-site binding

## Evidence

- Member CRUD/search/stats, member sessions, and member audit listing now use
  `NewTenantTransactor` / `WithinQuery`.
- `cmd/hydro` passes `RequireTenantScope` into the member audit store.
- `go test ./internal/database ./modules/members/sqlstore ./modules/members ./modules/members/transport` passed.

## Remaining scope

The direct-pool inventory is not complete for media, forms, transport
projections, and mobile-specific paths. No claim is made for live PostgreSQL
RLS until the shared/single-tenant drill runs against PostgreSQL.
