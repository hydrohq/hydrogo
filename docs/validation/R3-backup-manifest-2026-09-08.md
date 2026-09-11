# R3 Compact SQLite snapshot validation

**Date:** 2026-09-08  
**Status:** Snapshot/manifest, byte/time budget, and SQLite restore control state succeeded; clean environment restore and PostgreSQL gate remain open.

`SnapshotSQLite` within `internal/backup` takes an SQLite snapshot to the staging directory using `VACUUM INTO` assuming controlled write stop. Media files are copied under `media/` and their SHA-256/byte size can be written to the manifest. `Verify` validates database and media integrity without activating them; media modification and traversal/duplicate media manifest paths are rejected in the test. `RestoreToStaging` only succeeds on an empty target, copies after validation, and fails without session revocation, external egress quarantine, and a marker indicating that old writes are stopped.

Migration `202609090015_restore_controls` moves the restore state to the DB. `202609090016_restore_write_fence` adds a DB trigger to critical pages/media/news/jobs tables. `ApplyRestoreControls` revokes all sessions and activates the `restore_controls` row; it returns no claims while the job store quarantine is active, and the final atomic lease update of a claim re-checks quarantine/dead-letter conditions, with old deployment writes rejected by the trigger. After staging validation, `ReleaseRestoreQuarantine` opens job claims; after draining the old slot, `ReleaseRestoreWrites` removes the write fence. SQLite and isolated PostgreSQL schema contracts validate the same control sequence.

`SnapshotSQLiteWithLimits` checks media total, SQLite page estimate/snapshot byte size, and wall-clock budget before and during the staging copy. If the budget is exceeded, the new destination is cleaned and a usable restore marker is not produced. The assembled staging tree is re-validated right before the snapshot is published; if the source media changes during copying, the snapshot is wiped due to a checksum mismatch. `Verify` now enforces that the `manifest.json` record exists within the snapshot and matches byte-for-byte with the caller-validated manifest. The snapshot directory now carries its own validation input with `manifest.json`; the contract test checks that the returned manifest matches the one on disk. `PostgresTools` restricts the custom-format `pg_dump`/`pg_restore` call with a temporary `PGPASSFILE` without placing the password in process arguments; `SnapshotPostgres` and `RestorePostgresToStaging` validate manifest-verified media copy, quarantine marker, and isolated target database round-trip with CI client tools. `ActivateStagingMedia` does not run without a marker carrying the three checks; it atomically promotes the staging root to the active root and preserves the old root in a pre-emptied rollback directory.

`PruneSnapshots` only evaluates local snapshot directories carrying a valid `manifest.json`; it retains the newest snapshot count and pinned directories, stopping safely on a corrupt/partial manifest. This helper does not replace external object storage GC or erasure log retention coordination.

Executed check:

```text
go test ./internal/backup
go test ./...
```

This experiment does not add a restore command to the application CLI, nor does it prove media activation, actual traffic switchover after PostgreSQL restore, or true rejection of old binary writes. These are the subsequent steps of S07.
