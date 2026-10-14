# ADR-0079: Standardize frontend state and error semantics

## Status

Accepted

## Decision

Use the shared `idle`, `loading`, `success`, `empty`, and `error` state model
across Admin, public, member, and optional external frontend surfaces. Error
codes are stable and machine-readable; user messages are safe/localizable;
technical details remain operational telemetry. Optimistic behavior must be
explicit and reversible.

## Consequences

Users see predictable loading, empty, retry, and failure behavior, while teams
can reuse state-aware components. Product-specific copy and retry policy remain
owned by each surface within this contract.
