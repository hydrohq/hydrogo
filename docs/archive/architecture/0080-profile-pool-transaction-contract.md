# ADR-0080: Capacity profile pool and transaction contract

## Status

Accepted

## Context

HydroGo capacity profiles already expose bounded database pool budgets, but the
adapter contract did not prove that those values are applied consistently or
that the shared transaction helper preserves atomic commit and rollback
semantics.

## Decision

- Each capacity profile maps its `DBMaxOpenConns` and `DBMaxIdleConns` budget to
  `database.OpenWithOptions`.
- Local SQLite contract tests verify the mapping and atomic transaction
  behavior. The tests use only the public adapter boundary and do not expose a
  driver-specific detail to domain code.
- PostgreSQL lock behavior and statement timeout enforcement remain a
  dialect/production gate. They must be certified against the supported
  PostgreSQL version before production launch; SQLite tests must not be treated
  as that certification.

## Consequences

Compact deployments retain a one-connection default and do not gain hidden
workers or unbounded pools. Standard and Distributed profiles have explicit,
reviewable pool limits. Future adapters must preserve the `Transactor.Within`
  commit-on-success and rollback-on-error contract.

## Verification

See [`docs/validation/R5-profile-pool-transaction.md`](../validation/R5-profile-pool-transaction.md)
and `internal/database/profile_contract_test.go`.
