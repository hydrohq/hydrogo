# ADR-0129: News editorial transition policy

- **Status:** Accepted
- **Date:** 2026-09-16
- **Related decisions:** [ADR-0122](0122-versioned-editorial-workflow-contract.md), [ADR-0013](0013-identity-and-security-boundaries.md)

## Decision

The News application service accepts an optional versioned
`extension/v1.EditorialWorkflow`. If omitted, the existing draft/scheduled/
published/retracted transition graph is used. Schedule, publish and retract
validate the current persisted state inside the same transaction that performs
the mutation, preventing a policy check from racing a concurrent write.

The current database status contract remains unchanged. Persisted arbitrary
moderation states, role-specific transition authorization and workflow editing
in Admin Studio require a future migration and separate product decision.

## Evidence

`modules/news/workflow_test.go` proves a configured policy can reject a publish
transition without changing the stored status. The service still delegates
persistence, audit and outbox behavior to module-owned boundaries.
