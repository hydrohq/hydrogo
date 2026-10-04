# R33 — WSL production-gate preflight on current client SDK commit

**Date:** 2026-09-17  
**Commit:** `9f95384f070c8ac5429c4b96d33d9231aa7362e8`  
**Environment:** Ubuntu WSL2, Linux amd64, Go 1.27.0, PostgreSQL 18.6,
Redis/Valkey local service

## Command

```bash
cd /path/to/project
export HYDRO_TEST_POSTGRES_URL='postgres://user@localhost:5432/hydro_contract?sslmode=disable'
export HYDRO_TEST_REDIS_URL='redis://127.0.0.1:6379/15'
bash scripts/verify-production-gates.sh
```

## Result

The automated preflight completed successfully. The following gates passed:

- format, module integrity, workflow action pin and dependency scan;
- `go vet`, full and shuffled suites;
- PostgreSQL-backed race-sensitive packages;
- reproducible `hydro` build;
- portable export benchmark (`BenchmarkSQLitePortableRoundTrip256`:
  10,149,394 ns/op on this WSL runner);
- Redis/Valkey atomic limiter contract.

Credentials were supplied only through environment variables and are not in
this report. The disposable `hydro_contract` database contains no customer
data.

## Remaining external drills

This preflight does not close the real-environment gates for an empty-target
PostgreSQL restore, reverse-proxy two-process blue/green rollback, external
backup/key recovery, production signing-key ceremony/provenance publication,
or representative customer capacity/RPO/RTO measurements.
