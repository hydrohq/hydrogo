# 0189. Backup, Restore & Point-In-Time Failover Isolation Certification

Date: 2026-09-20
Status: Accepted

## Context

Enterprise data protection for HydroGo requires that backups and restore procedures operate with strict isolation, cryptographic manifest validation, session invalidation, and job queue write-fences. When restoring a database and its associated media assets:
1. Active user and member sessions from the restored snapshot must not persist into the target environment (all sessions must be revoked upon restore to prevent session hijacking and credentials replay).
2. Outbound asynchronous workers and background jobs from the restored snapshot must be fenced or marked quarantined so that stale webhook or email delivery events are not erroneously dispatched.
3. Media assets must be verified against SHA-256 cryptographic digests in the backup manifest before activation.
4. Retention policies must safely prune unpinned snapshots while preventing symlink traversal or arbitrary file deletion outside the backup destination root.

## Decision

1. **Restoration Controls & Invalidation (`internal/backup/controls.go`):**
   - Restorations must invoke `EnforceRestoreControls` across SQLite and PostgreSQL, executing session purge (`DELETE FROM sessions`) and job quarantine fences (`UPDATE jobs SET status = 'quarantined'`).
   - Rejection of unsafe file traversal and symlink roots during staging activation and restore path validations.

2. **Immutable Media Manifests & Validation (`internal/backup/sqlite.go`, `internal/backup/postgres.go`):**
   - Backups generate a cryptographically signed/hashed `manifest.json` cataloging database checksums, schema version, and media asset digests.
   - Restorations fail closed immediately if any asset digest does not match its manifest entry or if manifest tampering is detected.

3. **Safe Snapshot Retention (`internal/backup/retention.go`):**
   - Provide non-mutating retention planning (`PlanSnapshotPrune`) and atomic execution (`PruneSnapshots`) ensuring retention thresholds preserve the newest snapshots and pinned backups.

## Consequences

- **Positive:** Guarantees zero data pollution or credential leakage during disaster recovery, staging migrations, and PITR drills.
- **Positive:** Fail-closed guarantees protect filesystem integrity against directory traversal vulnerabilities.
- **Operational:** Restores require users to re-authenticate, eliminating post-restore session hijack risks.
