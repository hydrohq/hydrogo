# R12 WSL production-gate preflight (2026-09-17)

## Result

On Ubuntu WSL2, `scripts/verify-production-gates.sh` completed successfully
with the isolated PostgreSQL 18.6 database and local Redis/Valkey service at
commit `4a8d999`.

Passed automated gates:

- formatting, module integrity, action pin and dependency vulnerability scan,
- `go vet`, full and shuffled test suites,
- race-sensitive package suite against the PostgreSQL URL,
- reproducible Hydro binary build,
- portable SQLite benchmark,
- Redis/Valkey atomic limiter and TTL contract.

The report contained no credentials and was generated under `.tmp/gates`.

## Remaining external drills

The preflight intentionally does not certify customer-like clean-target restore,
reverse-proxy two-process traffic switching/rollback, Redis failover and
partition/clock-skew behavior, external backup/key recovery, signing-key
ceremony, payment/provider integrations, or customer RPO/RTO/capacity/soak
measurements. Those require isolated staging infrastructure and operator-owned
evidence.
