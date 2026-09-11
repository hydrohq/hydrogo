# R3 migration restart and compatibility validation

**Date:** 2026-09-08  
**Roadmap:** R3 / S06 foundation gate  
**Status:** SQLite restart/checksum and expand/rollback compatibility foundation gates succeeded; actual blue/green traffic and full profile measurement remain open.

All migrations were applied on a clean SQLite file and the `schema_migrations` ledger was validated. After closing and reopening the DB, the second `ApplyMigrations` call was a no-op; the number of migration rows did not change. When the ledger checksum was simulated as altered, `CheckCompatibility` refused to start the service. A ledger record applied by a newer binary but not found in the current binary's embedded migration set is now rejected fail-closed with `ErrUnknownMigration`; thus, it does not appear compatible for an old binary to roll back to a database with an advanced schema. The backup staging test also proves that a retained media file is restored with the exact same byte content after validation.

```text
go test ./internal/database
```

This test does not solely certify old binary and new write/rollback compatibility, online DDL lock behavior, or actual slot traffic.

## S06 progress validation — 2026-09-09

After applying the full migration set, `tests/contract/migration_rollback_test.go` validates in SQLite that the new binary writes a rich page projection, the old binary reads and updates only the `id/title/version` columns, and the new binary reads preserving the new fields. Under the CI condition where `HYDRO_TEST_POSTGRES_URL` is defined, the same contract also runs on PostgreSQL.

`internal/release.RetainAssets` verifies the `static/`, `assets/`, and `public/assets/` artifacts from the release manifest via checksum and atomically moves them to the shared retention directory; if the existing asset at the same path differs, the change is rejected, and `VerifyRetainedAssets` re-checks integrity during the rollback window.

`hydro upgrade check` checks config/profile, DB ping, and migration ledger checksum compatibility without running migrations; a successful preflight outputs `upgrade-ready` and does not open readiness upon a schema mismatch.

The migration runner now respects the `transactional.<dialect>` value in the metadata. Sequential migrations with a transaction remain in the same transaction group; a `transactional: false` migration runs with its own connection execution and ledger record. The SQLite `VACUUM` fixture validates this distinction at the contract level. Under the CI condition with `HYDRO_TEST_POSTGRES_URL` defined, the `CREATE INDEX CONCURRENTLY` migration is applied outside a transaction while repeated short read queries remain successful and the index becomes visible. It must be separately measured whether PostgreSQL long-lived snapshot/read transactions can wait for concurrent indexes; actual slot rollback is still a separate S06 drill.

`internal/release.Deployment` enforces the ADR-0005 sequence as a local state machine: Green is prepared, readiness is obtained, retained asset warm-up is completed, schema rollback safety is confirmed, then switch/drain/rollback is performed. This contract does not solely certify actual reverse-proxy traffic or process restarts.

`internal/release.Traffic` atomically switches the process-local HTTP handler to Green after the same gate and returns it to Blue on a safe rollback. This contract does not replace external reverse-proxy integration and process restart drills.

The `httptest` contract with reverse proxy handlers validates that the actual `httputil.ReverseProxy` flow reaches the Green backend after the gated switch. Production proxy configuration and process restart drills remain open.

The PostgreSQL contract also validates that the `CREATE INDEX CONCURRENTLY` operation does not finish immediately while a long-lived snapshot transaction is open, and that the migration completes after the snapshot is committed. This measures lock behavior; the production time budget and actual rollback drill remain open.

`TestSQLiteOnlineExpandAllowsConcurrentReadContract` validates that a nullable expand migration completes within a two-second deadline while an active read transaction is ongoing in WAL-enabled SQLite, and the new column is visible. This experiment does not certify the overall production lock time of the migration or the PostgreSQL DDL plan; actual blue/green traffic, production rollback after new writes, and capacity measurement are the open parts of S06/S07.
