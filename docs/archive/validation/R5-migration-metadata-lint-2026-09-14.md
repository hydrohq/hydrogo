# R5 migration metadata lint validation — 2026-09-14

- `go test ./tools/migrationguard` passes.
- The guard requires sqlite/postgres transactionality, checksums, lock-timeout,
  disk estimate, compatibility, and rollback fields.
- Existing migrations pass the stricter metadata contract in local verification.

Production lock measurements and rollback drills remain external gates.
