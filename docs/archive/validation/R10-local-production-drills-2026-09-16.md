# R10 local production-drill evidence (2026-09-16)

## Result

`scripts/verify-local-production-drills.ps1` completed successfully on the
Windows development host at commit `3490044`.

Passed drills:

- isolated SQLite binary migration plus `/healthz` and `/readyz` (HTTP 200),
- backup, release, and portable contract packages,
- blue/green cache rollback stability repeated 20 times.

The generated timestamped report is intentionally kept under `.tmp/gates` so
it does not become a mutable source artifact.

## Explicitly pending

`HYDRO_TEST_POSTGRES_URL` was not configured for this run, so the PostgreSQL
contract/restore drill remains pending. Reverse-proxy traffic switching,
external backup/key recovery, production signing-key ceremony, and
customer-workload RPO/RTO/capacity measurements still require their respective
isolated environments and operator evidence.

This report is local evidence only; it does not claim production certification.
