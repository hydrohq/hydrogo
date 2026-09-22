# ADR-0236: Bounded Admin bulk execution primitive

- **Status:** Accepted
- **Date:** 2026-09-22
- **Related decisions:** [ADR-0104](0104-bounded-admin-bulk-contract.md), [ADR-0106](0106-opt-in-admin-bulk-list-surface.md)

## Decision

`internal/adminstudio.ExecuteBulk` provides a bounded, deterministic,
sequential orchestration primitive for module-owned bulk mutations. It
normalizes and sorts IDs, limits the selection to the caller-provided value
(never above 500), returns per-item success/failure codes without exposing raw
errors, and stops before the next item when the request context is cancelled.

The callback remains responsible for authorization, tenant scope, idempotency,
audit events, domain transitions and transaction semantics. Core does not add
parallel workers, retries, or an implicit transaction because those choices
would amplify resource usage and could violate a module's consistency model.

## Consequences

- Editions can expose a consistent partial-failure response without copying
  normalization logic into every mutation endpoint.
- Sequential execution is predictable and safe for the Compact profile, while
  a module may wrap the callback in its own transaction or queue contract.
- A successful response with `Completed: false` is explicitly distinguishable
  from a completed operation; cancelled work must be retried with the module's
  idempotency policy.

## Verification

`internal/adminstudio/bulk_test.go` covers deterministic ordering, callback
error redaction, per-item accounting and cancellation boundaries.
