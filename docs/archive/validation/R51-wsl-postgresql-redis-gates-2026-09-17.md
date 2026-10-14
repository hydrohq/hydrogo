# R51 — WSL PostgreSQL + Redis production-gate preflight

**Date:** 2026-09-17  
**Commit:** `c7eb10ff745363fecee184508146aa380593a0a3`  
**Status:** Successful (local service evidence)

`scripts/verify-production-gates.sh` was executed against PostgreSQL 18.6 and Redis/Valkey services running on Ubuntu/WSL2. The test DSN was used strictly over local Unix socket; no passwords or tokens were written to the report.

Passing gates:

- gofmt, module integrity, workflow action pins, and govulncheck
- go vet, full test suite, and shuffled test suite
- race-sensitive package suite (including PostgreSQL contract tests)
- reproducible HydroGo build
- SQLite portable benchmark
- Redis/Valkey atomic limiter + TTL test

Runner report: `.tmp/gates-wsl/production-gates-20260917-155008.md`.

This evidence does not replace restore with real customer data, reverse-proxy blue/green/rollback, external backup/key recovery, production key ceremony, or customer capacity/RPO/RTO drills; these remain open production gates.
