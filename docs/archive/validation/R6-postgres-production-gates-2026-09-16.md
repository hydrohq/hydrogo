# PostgreSQL production-gate evidence (2026-09-16)

An isolated `hydro_gate_local` database was used with PostgreSQL 18.6 running on WSL Ubuntu. The connection utilized peer authentication; no passwords or secrets were written to the report. The temporary database should be dropped post-test.

## Command

```bash
cd /path/to/project
HYDRO_TEST_POSTGRES_URL="postgres://user@localhost:5432/hydro_gate_local" \
  HYDRO_GATE_REPORT=.tmp/gates/postgres-local-20260916-full.md \
  bash scripts/verify-production-gates.sh
```

## Result

Full flow **PASS**:

- gofmt, module integrity, workflow action pins, govulncheck, and go vet;
- full and shuffled test suite including SQLite/PostgreSQL;
- race-sensitive packages;
- reproducible HydroGo build yielding identical hashes twice;
- portable benchmark (`10.3 ms/op`, 10 iterations).

This execution provides PostgreSQL connectivity/contract and local automated gate evidence. Restore to a clean target with real customer data, proxy blue/green rollback, external backup/key recovery, and capacity/RPO/RTO drills remain production gates.
