# R74 — WSL automated production-gate preflight (2026-09-19)

## Scope

This evidence records the automated preflight for commit `7b2b516b6eb42beb47a4f30d25ad61131d6994fe` using the compact profile under WSL2. It is a local/CI-style verification record, not proof of a production deployment or customer certification.

## Result

`bash scripts/verify-production-gates.sh` completed successfully.

Verified gates:

- formatting, module integrity, and workflow action pin checks
- dependency vulnerability scan and `go vet`
- full and shuffled Go test suites
- race-sensitive package tests
- reproducible Hydro build
- portable benchmark
- Redis/Valkey atomic limiter test

The configured PostgreSQL contract target was used with credentials omitted from the report.

## Explicitly still open

The following require real environment/operator evidence and are not covered by this report:

- PostgreSQL restore into a clean target
- reverse-proxy two-process blue/green restart and rollback
- external backup and signing-key recovery
- production key ceremony and provenance publication
- representative customer capacity, RPO, and RTO measurements

Source output: `.tmp/gates/r74-production-gates-20260919.md` (generated locally and intentionally untracked).
