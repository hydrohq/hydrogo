# ADR-0234: Generated Admin Studio view metadata

- **Status:** Accepted
- **Date:** 2026-09-22
- **Related decisions:** [ADR-0231](0231-bounded-admin-saved-view-contract.md), [ADR-0229](0229-generated-list-transport-contract.md)

## Decision

Every generated resource emits `admin/views.json` with a deterministic default
view derived from reviewed presentation metadata. Visible list fields become
columns and filterable fields become bounded filter descriptors. The artifact
contains metadata only; it never executes a query, stores user preferences or
grants field access.

User-specific saved views may extend this metadata only after the public
AdminView contract, resource query parser, authorization and tenant scope have
validated them. The generator lock and compatibility checker treat the file as
a required generated artifact.

## Verification

Generator fixtures verify the artifact exists and compiles alongside the
resource contract. Editions must add persistence ownership, CSRF, optimistic
concurrency and policy tests before exposing saved-view mutations.
