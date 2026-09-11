# ADR-0005: Blue/green and zero-downtime release transition

- **Status:** Superseded (partially)
- **Date:** 2026-09-05
- **Superseded by provisions in:** [ADR-0011](0011-consistency-upgrade-and-recovery.md)

> Blue/green, process roles, proxy, and expand-migrate-contract approach remain valid. Warm-up/invalidation, contract testing in production, schema/job/content compatibility, and restore details are based on ADR-0011. Zero downtime is a product goal; it is not a realized guarantee without providing the certified topology and tested release conditions.

## Context

HydroGo customers should not experience downtime during content publication; at the same time, database migrations and background jobs must not corrupt data integrity.

## Decision

HydroGo supports blue/green deployment for public web traffic. The zero-downtime guarantee is provided only for `online` migrations that are backward compatible with the old and new binary. Backward-incompatible table transformations require a controlled maintenance window.

## Process roles

The same binary can run with different roles:

```text
hydro serve --role=all
hydro serve --role=web
hydro serve --role=worker
```

- Compact: `all`
- Blue/green: Two web slots and a single active worker
- Distributed: Multiple web and multiple workers using lease

## Single server blue/green flow

1. The new release is downloaded to the passive slot and its signature is verified.
2. `hydro upgrade check` verifies the config, disk, DB capability, and migration class.
3. A backup/snapshot is taken before deployment.
4. The `online` expand migration is applied; a migration lock prevents concurrent execution.
5. Green web starts with the worker off and on a different port/socket.
6. Readiness, DB contract, and internal smoke tests run.
7. Critical public paths are called to warm up the L1 cache.
8. The reverse proxy atomically switches traffic to Green.
9. Blue stops receiving new requests and completes its current requests.
10. The worker restarts with the new release, or leadership passes to Green.
11. Blue is kept as the previous release for the rollback period.

Caddy is the default single-server reverse proxy solution because its config reload can load the new configuration without stopping the old one. The Admin API is protected only over the local socket/interface.

## Migration classes

### Online

The old and new binary can work together on the same schema. Generally, it is the preparation for a new table, nullable column, or a backward-compatible index/constraint.

### Backfill

The schema is expanded; data is migrated in small, restartable chunks. During the code transition period, it can read/write the old and new representations.

### Maintenance

It is a table rewrite, a backward-incompatible constraint, column removal, or a transformation that the old binary cannot read. Deployment is carried out with a maintenance page and an explicit RTO.

## Expand–migrate–contract

Breaking changes are spread across releases:

1. **Expand:** The new structure is added, the old structure is kept.
2. **Migrate:** Data is migrated in the background and verified.
3. **Switch:** The new code uses the new structure as the primary source.
4. **Contract:** The old structure is removed only after the support window ends.

## Database profiles

- SQLite: Blue and Green can briefly co-exist on the same machine and local disk; the single writer at a time limit is preserved.
- PostgreSQL: Multiple web instances and rolling/canary deployments can be supported.
- Future certified adapters must also pass online migration and multi-instance contract tests.
- Online DDL is actually verified for each driver; the "online" tag is not given solely by looking at SQL syntax.

## Session, media and assets

- Session is not a single source of truth in process memory; since it is kept in the DB, the user is not logged out upon slot switch.
- Media is immutable and resides in a storage accessible by both slots.
- Asset URLs carry a content hash.
- The new release serves assets referenced by the old cached HTML for the duration of the cache lifetime.

## Cache consistency

- In atomic slot transition, the Green cache is pre-warmed.
- Versioned cache keys are used in the short Blue/Green overlap.
- If there is canary or long-term concurrent traffic, L2 distributed cache/invalidation is mandatory.

## Rollback

The proxy returns to the previous slot only if the current schema is within the compatibility range declared by the previous binary. If the migration is backward-incompatible, automatic binary rollback is not performed; controlled restore or roll forward is applied.

## Guarantee

HydroGo uses the following statement:

> Backward-compatible releases are deployed without interruption to public traffic. Structural data transformations are applied with a safe and pre-announced maintenance window.

## References

- [Caddy zero-downtime configuration reload](https://caddyserver.com/docs/getting-started#reloading-config)
