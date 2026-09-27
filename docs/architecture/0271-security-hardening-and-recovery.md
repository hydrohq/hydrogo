# ADR-0271: Security hardening and recovery controls

- **Status:** Accepted
- **Related:** [ADR-0011](0011-consistency-upgrade-and-recovery.md), [ADR-0013](0013-identity-and-security-boundaries.md), [ADR-0030](0030-client-address-trust-boundary.md), [ADR-0060](0060-migration-metadata-lint.md), [ADR-0263](0263-sqlite-pragma-profiles.md)
- **Date:** 2026-09-28

## Context

HydroGo needs security controls that remain usable on compact SQLite hosting
without requiring Redis, an external WAF, or a system utility. Controls must
keep database and file recovery explicit, bound in-memory state, and preserve
SQLite's WAL consistency rules.

## Decision

- Authentication failures use a bounded, expiring in-memory tarpit. Successful
  authentication clears the corresponding address/account entry. TOTP and
  hashed one-time recovery codes use the existing MFA storage contract.
- IP and CIDR bans are persisted in the SQL store and loaded into a bounded
  in-memory matcher; the CLI and administrator surface update that matcher
  immediately. Proxy-derived addresses continue to follow ADR-0030.
- HTTP security headers use named strict, standard, and relaxed presets.
  Obvious injection markers may be rejected by a bounded defense-in-depth
  filter; applications must still validate input and use parameterized SQL.
- Uploads check available disk space before writing, validate content from
  bytes rather than the filename, reject executable double extensions, and
  remove JPEG/WebP EXIF metadata from decoded images.
- SQLite startup recovery runs `quick_check` and a restart checkpoint. It never
  unlinks WAL/SHM files because committed data may still reside in the WAL.
- SQLite migrations create a timestamped `VACUUM INTO` snapshot before DDL.
  `hydro rollback` requires operator confirmation and a stopped runtime; it
  supports SQLite only and preserves the replaced database beside the active
  path. PostgreSQL recovery remains dump/restore tooling.
- Existing audit events are append-only at the database layer. Security
  rejections are recorded through the runtime audit recorder.

## Compatibility and operations

The IP-ban and audit immutability migrations are additive and module-scoped.
The audit trigger migration runs only when the pages module (which owns the
existing audit table) is enabled. Quota checks use the local media filesystem;
remote storage drivers retain their own capacity policy. The in-memory ban
refresh runs only in persistent server mode and is bounded to 4096 entries.

Operators can inspect `hydro migrate --dry-run` before applying migrations.
Migration snapshots and rollback copies consume additional disk space, so
deployment capacity must reserve room for one database-sized snapshot.

## Consequences

The compact profile needs no new service. Tarpit and active ban matching are
process-local; persisted bans repopulate them after restart. The anomaly filter
is heuristic and can have false positives, so trusted editor paths can be
explicitly bypassed. This filter is not a substitute for application security.

## Verification

SQLite and PostgreSQL migration metadata includes expand compatibility,
rollback/recovery notes, and smoke checks. Unit tests cover tarpit expiry,
ban matching and persistence, disk quota, upload sniffing, image metadata
removal, anomaly filtering, SQLite recovery, and snapshot behavior.
