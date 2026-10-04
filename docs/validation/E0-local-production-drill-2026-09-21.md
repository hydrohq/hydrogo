# E0 Local Production Drill — 2026-09-21

**Commit:** `fc534692`  
**Status:** Local binary/readiness and contract gates passed; external deployment gates remain open.

## Windows isolated drill

`scripts/verify-local-production-drills.ps1` passed:

- isolated SQLite build, migration, `/healthz`, and `/readyz` smoke;
- backup, release, and portable contract suites;
- 20-run concurrent blue/green cache rollback stability test.

The Windows script records PostgreSQL as pending because the WSL socket is not
exposed to the Windows process. The equivalent WSL run used a fresh database
and passed:

```text
HYDRO_TEST_POSTGRES_URL=postgres://user@localhost:5432/hydrogo_backup_cert2 \
  go test ./internal/backup ./tests/contract -count=1 -timeout=5m
ok  github.com/ardaozyildirim/hydrogo/internal/backup
ok  github.com/ardaozyildirim/hydrogo/tests/contract
```

The aggregate runner now supports `HYDRO_RUN_WSL_POSTGRES=1`; the same command
completed SQLite, backup/release/portable, cache rollback, and disposable WSL
PostgreSQL drills in one report. The temporary database was dropped after the
run.

With both `HYDRO_RUN_WSL_POSTGRES=1` and `HYDRO_RUN_WSL_REDIS=1`, the aggregate
runner additionally passed the disposable Redis distributed-admission/race
drill and cleaned up both temporary services.

## Still external

This does not certify a real reverse-proxy/process restart rollback, external
backup or key escrow recovery, production signing-key ceremony, or customer
workload capacity/RPO/RTO. Those require a controlled staging environment and
operator evidence; they are intentionally not marked complete by this local
drill.
