# HydroGo Upgrade & Migration Runbook

This document describes zero-downtime upgrades, migration safety, and rollback procedures for HydroGo Core and Client Edition projects.

---

## 1. Pre-Upgrade Preflight (`hydro upgrade check`)

Before advancing to a new Core version, run the `upgrade check` command to verify that the current database schema and configuration are ready for the new version:

```powershell
hydro upgrade check --config config.yaml
```

Example output:
```text
upgrade-ready profile=compact database=sqlite
```

### Checks Performed:
1. Database connectivity and reachability (`PingContext`).
2. Ledger compatibility of embedded Core migrations (`database.CheckCompatibilityForModules`).
3. Verification that the schema version contains no irreconcilable conflicts.

---

## 2. Lockfile (`hydrogo.lock`) and Core Compatibility

During each upgrade operation, `hydrogo.lock` is inspected:
- **Core Version:** Specifies the exact Core version against which the customer project was tested.
- **Generator Version:** The generator version used to emit the scaffold.
- **Features:** Catalog of enabled modules.

When Core is upgraded, version metadata inside `hydrogo.lock` is updated in a controlled, deliberate manner.

---

## 3. Additive Migration Principle

- HydroGo never emits destructive DDL such as `DROP TABLE` or `DROP COLUMN` at runtime or in automated migrations.
- To ensure that older and newer binaries can safely operate side by side during blue/green cutovers (ADR-0005), all schema migrations are strictly additive.
- New columns are added as `NULL` or with explicit `DEFAULT` constraints.

---

## 4. Rollback Procedure

If an issue is detected with the new release:
1. Immediately re-route traffic back to the previous binary.
2. Additive migrations do not prevent the previous binary from operating normally.
3. If necessary, follow snapshot restore procedures from the latest certified backup:
   ```powershell
   hydro restore verify --snapshot /backups/2026-09-12
   hydro restore stage --snapshot /backups/2026-09-12 --destination /data/staging --sessions-revoked --egress-quarantined --old-writes-blocked
   hydro restore activate-media --staging /data/staging/media --active /data/media --rollback /data/media_old
   ```
