# ADR-0036: Bounded public catalog projections

- **Status:** Superseded by ADR-0199
- **Date:** 2026-09-13
- **Related decisions:** [ADR-0007](0007-capacity-profiles.md), [ADR-0011](0011-consistency-upgrade-and-recovery.md), [ADR-0017](0017-commerce-consistency-and-payments.md)

## Decision

Public product and variant projections use a bounded page size of 100 records
and an explicit page offset. REST consumers may request subsequent pages with
the `page` query parameter; GraphQL and HTML projections use the first bounded
page until their own cursor contract is introduced. Administrative full-list
reads remain a separate privileged projection.

## Consequences

Public requests cannot force an unbounded product result into application
memory. A catalog UI with more than 100 products must add navigation before all
records are expected to be visible. The SQL dialect remains an implementation
detail of the repository adapter, and no migration is required.
