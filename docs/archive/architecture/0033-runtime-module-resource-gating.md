# ADR-0033: Runtime module resource gating

- **Status:** Accepted
- **Date:** 2026-09-12
- **Related decisions:** [ADR-0010](0010-sdk-lifecycle-and-edition-manifest.md), [ADR-0011](0011-consistency-upgrade-and-recovery.md)

## Decision

Runtime resources are started only when an active module can produce or consume
their work. The shared outbox dispatcher starts only if Pages, News, or
Commerce is active. The integrated root-page redirect is registered only with
the Pages module.

## Consequences

Headless or minimal builds no longer run an idle outbox goroutine or point `/`
at an absent public page. This is an additive runtime-composition correction:
stored outbox records remain intact, and reactivating a producer module resumes
delivery through the normal release process. No migration or cache change is
required.
