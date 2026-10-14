# R8 local production drills — 2026-09-10

**Status:** Local SQLite and PostgreSQL drills passed. This report is evidence
for a reproducible developer-host drill; it is not a production certification
or an RPO/RTO/SLA claim.

## Environment

- Host: Windows 11, `Microsoft Windows NT 10.0.26100.0`
- Go: `go1.27.0 windows/amd64`
- PostgreSQL: local PostgreSQL 18 service on `127.0.0.1:5432`
- Database: dedicated `hydro_contract` database and `hydro` test role
- Commit: `354bec1024206acf9ae44596844eb3c9f50dfdd6`

Credentials were supplied through environment variables and are not stored in
this report or repository.

## Commands and results

The reproducible local runner was:

```powershell
$env:HYDRO_TEST_POSTGRES_URL = "postgres://hydro:...@127.0.0.1:5432/hydro_contract?sslmode=disable"
$env:HYDRO_PG_DUMP = "C:\Program Files\PostgreSQL\18\bin\pg_dump.exe"
$env:HYDRO_PG_RESTORE = "C:\Program Files\PostgreSQL\18\bin\pg_restore.exe"
powershell -ExecutionPolicy Bypass -File .\scripts\verify-local-production-drills.ps1
```

The runner passed all enabled local drills:

- isolated SQLite binary build, migration, server start, `/healthz`, `/readyz`,
  and shutdown;
- backup, release, and portable contract packages;
- blue/green cache rollback stability repeated 20 times;
- PostgreSQL backup/restore and `tests/contract` packages against a clean
  HydroGo test database.

The generated evidence report is written under `.tmp/gates/` and intentionally
does not enter version control because it contains host-specific details.

## Remaining non-local evidence

The following cannot be honestly closed on this developer host:

- two real HydroGo processes behind the supported reverse proxy, including
  restart and rollback after writes;
- external backup storage and separately controlled key escrow recovery;
- production signing-key ceremony and external provenance publication;
- representative customer workload capacity, RPO, and RTO measurements.

These remain release gates in [`PRODUCTION-GATES.md`](PRODUCTION-GATES.md) and
are not replaced by this local drill.
