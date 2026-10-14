# Sectoral content modules operations

## Enablement and migrations

Enable only the collections used by an Edition. The migration runner filters
manifests by enabled module, so an inactive collection has no table, route,
asset, or worker. Apply its SQLite or PostgreSQL migration before deploying a
binary that writes that collection. Each table is isolated by module and
tenant; PostgreSQL additionally enforces tenant RLS.

## Capacity and runtime

Collection reads return at most 200 records and individual serialized payloads
are limited to 1 MiB. Use pagination for larger public collections. Writes
share the configured database pool and commit audit records in the same
transaction. There are no module-specific workers or cache dependencies.
Downloads increment their counter atomically. Event calendar exports use the
bounded published-event query.

## Rollback and recovery

Before removing an enabled collection, take the normal database backup and
confirm the Edition no longer needs its records. Use the migration rollback
metadata for that collection only; do not drop another module's table. Keep the
backup through the deployment rollback window. SQLite uses short write
transactions; PostgreSQL uses the runtime's tenant transaction settings and
RLS policies.

## Deployment compatibility

Roll out additively: migrate first, then deploy the new application version.
Older binaries ignore the added module tables. Keep module enablement consistent
across blue/green instances during the rollout so only instances with the
matching module contract receive its requests.
