# ADR-0227: Client Edition lifecycle hook extension contract

- **Status:** Accepted
- **Date:** 2026-09-22
- **Related decisions:** [ADR-0210](0210-blueprint-scaffold-boundary.md), [ADR-0226](0226-versioned-field-validation-contract.md)

## Decision

`hydro generate scaffold` creates an optional human-owned hook skeleton under
`custom/` for each resource. The generated package remains machine-owned and
contains no customer business rules. Hooks expose explicit Before/After points
for create, update and delete operations; implementations must keep
authorization, tenant scope and transactional decisions in the owning service.

Hooks must not perform network I/O inline with a database mutation. Durable
side effects are dispatched through the Edition's outbox contract so retries,
idempotency and auditability remain explicit. The scaffold is additive and
never overwrites an existing custom hook file.

## Verification

The CLI package compiles and its scaffold verification compiles both generated
and custom trees. Existing custom hook files are preserved. Editions must add
contract tests when a hook changes persistence, authorization, tenant scope or
external side effects.

Implementation order and exit gates are tracked in the [Roadmap](../ROADMAP.md).
