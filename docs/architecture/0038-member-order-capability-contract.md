# ADR-0038: Member order capability contract

- Status: Accepted
- Date: 2026-09-13
- Supersedes: none
- Related: [ADR-0037](0037-strict-modular-monolith-boundaries.md), [ADR-0029](0029-commerce-customer-capability-boundary.md)

## Context

The Members HTTP surface only needs a private, bounded customer-order
projection. It previously imported Commerce domain interfaces directly and
performed a type assertion, making a presentation module depend on another
module's implementation types.

## Decision

Members depends on the neutral `internal/contracts/customerorders.Reader`
capability. Commerce owns the adapter that maps its richer order model to this
projection, and the composition root wires that adapter. The projection omits
payment-customer secrets and remains bounded by the caller-provided limit.

## Consequences

- Members can be tested and deployed without importing Commerce domain types.
- Commerce remains the owner of order persistence and mapping rules.
- No schema or migration changes are required; SQLite and PostgreSQL behavior
  is unchanged.
- Future order providers can implement the contract without changing Members.
