# R23 — Forms and media RLS call-site binding

## Evidence

- Raw `*sql.DB` calls in the forms registry now enter
  `NewTenantTransactor`/`WithinQuery`.
- Media asset and variant operations now bind PostgreSQL tenant settings when
  called with a pool; existing transaction callers remain single-scope.
- Passed: `go test ./internal/database ./internal/forms/sqlstore ./internal/media/sqlstore ./internal/media ./modules/site ./modules/site/repository/sqlstore`.

## Remaining scope

Transport projections, commerce/mobile paths, and a live PostgreSQL
cross-tenant isolation drill remain open.
