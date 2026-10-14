# E0 WSL PostgreSQL Drill Runner — 2026-09-21

`scripts/verify-wsl-postgres-drill.ps1` now creates a uniquely named,
disposable PostgreSQL database inside WSL, runs the backup and contract suites
with the peer-authenticated `testuser` role, and drops the database in `finally`.

Observed pass:

```text
ok  github.com/ardaozyildirim/hydrogo/internal/backup
ok  github.com/ardaozyildirim/hydrogo/tests/contract
PASS: WSL PostgreSQL backup/contract drill
```

The runner was re-executed after the combined local drill and again passed on
the disposable database `hydrogo_backup_drill_20260921221957`; the database
was dropped during cleanup. A transient failure recorded by the aggregate
wrapper was therefore not treated as certification evidence.

The runner does not alter a shared database and does not claim production
privileged-role or external backup-key recovery certification; those remain
separate operational gates.
