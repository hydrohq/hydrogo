# R6 local production drills

**Commit:** `3badeba67f49ae1bf6200e9213680928fd09b1f5`  
**Environment:** Windows 10.0.26100, Go 1.27.0, amd64  
**Command:** `powershell -ExecutionPolicy Bypass -File scripts/verify-local-production-drills.ps1`

## Passed locally

- Isolated SQLite binary migration, startup, `/healthz`, and `/readyz` smoke.
- Backup, release-manifest, and portable export/import contract packages.
- Blue-green cache invalidation/rollback stability (`-count=20`).

## Explicitly pending

PostgreSQL execution was not claimed because `HYDRO_TEST_POSTGRES_URL` was not
configured. Reverse-proxy two-process rollback, external backup/key escrow,
production signing-key ceremony, and customer workload capacity/RPO/RTO remain
operator/environment gates.
