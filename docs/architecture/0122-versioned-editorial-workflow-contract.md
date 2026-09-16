# ADR-0122: Versioned editorial workflow contract

- **Status:** Accepted
- **Date:** 2026-09-16
- **Related decisions:** [ADR-0010](0010-sdk-lifecycle-and-edition-manifest.md), [ADR-0014](0014-publication-routes-and-erasure-recovery.md)

## Decision

Client Editions may declare bounded editorial states and transitions through
`extension/v1`. The SDK validates graph shape and transition membership but does
not own persistence, scheduling, authorization, audit, or publication timing.
Preview paths are constrained to same-origin paths; token signing and expiry
remain the host application's responsibility.

## Evidence and consequences

`extension/v1/workflow_test.go` covers valid/invalid transitions and external
preview-target rejection. Module integrations must add migration and workflow
policy tests before enabling custom states in production.
