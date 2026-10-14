# R3 local PostgreSQL 18.6 contract and restore validation

**Date:** 2026-09-09  
**Commit:** `7d557324d50e010146580c167d22f9b227845044`  
**Status:** Succeeded; local contract evidence, not commercial production certification.

## Environment

- PostgreSQL 18.6, Windows x86_64, `127.0.0.1:5432`
- Go toolchain: same `go 1.27.0` target as repository CI
- Test database: local database isolated for contract tests
- PostgreSQL dump/restore tools: PostgreSQL 18 client tools

## Executed gates

```powershell
$env:HYDRO_TEST_POSTGRES_URL = "postgres://...@127.0.0.1:5432/postgres?sslmode=disable"
go test ./internal/backup -run '^TestPostgresRestore' -count=1 -v
go test ./tests/contract -run '^TestPostgres' -count=1 -v
```

Successful evidence:

- Post-restore session revocation, egress quarantine, and old write fence.
- Inability to claim jobs without removing the quarantine; ability to claim after controlled opening.
- Pages/media erasure replay idempotency.
- Auth, media, News, Pages, Site, and transaction contracts.
- PostgreSQL concurrent migration runner and long-snapshot `CREATE INDEX CONCURRENTLY` behavior.

## Limits

This work was not performed on an actual customer database and does not prove external backup provider, key escrow, reverse proxy/process restart, production signing key ceremony, or customer RPO/RTO measurement. S07 sales certification and clean, external target restore drill remain open.
