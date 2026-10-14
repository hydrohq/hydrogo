# ADR-0231: Bounded Admin Studio saved-view contract

- **Status:** Accepted
- **Date:** 2026-09-22
- **Related decisions:** [ADR-0228](0228-public-resource-query-contract.md), [ADR-0230](0230-bounded-relation-plan-contract.md)

## Decision

Admin Studio saved views are versioned metadata, not executable queries. A
view may select allowlisted columns, bounded filters and bounded sort fields.
The contract validates resource field/operator metadata and value size before a
view is persisted. Column order is preserved because it is part of the editor
experience.

Before execution, the Edition must convert filters through the public resource
query parser and apply authorization, tenant scope and rate limits. Core does
not persist user-specific views or infer policy from a view definition.

## Verification

Public extension tests cover identity, duplicate/unknown columns, per-field
operator restrictions and filter budgets. Admin implementations must add
tenant ownership, CSRF and optimistic-concurrency tests around persistence.
