# R105 — WSL PostgreSQL/Redis production-gate preflight

**Date:** 2026-09-22  
**Commit:** `95fce7a76323b5ca5af31473bba770d3f13a0683`  
**Status:** PASS (WSL2 local services)

`scripts/verify-production-gates.sh` passed the following gates on PostgreSQL 18 and Redis/Valkey 8:

- gofmt and `go mod verify`
- workflow action pin checks
- govulncheck (no vulnerabilities in symbols called by code; transitive uncalled dependency notices reported in tool output)
- `go vet ./...`
- full and shuffled test suite
- race-sensitive packages
- reproducible HydroGo binary build
- portable SQLite benchmark
- Redis/Valkey atomic limiter + TTL test

This evidence does not cover restore to an empty PostgreSQL target, two-process reverse-proxy blue/green rollback, external backup/key recovery, production key ceremony, or customer RPO/RTO/capacity measurement. The gate runner intentionally leaves these marked as `EXTERNAL DRILLS STILL REQUIRED`.
