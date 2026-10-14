# ADR-0057: Evidence-based release checklist

- **Status:** Accepted
- **Date:** 2026-09-14
- **Related decisions:** [ADR-0010](0010-sdk-lifecycle-and-edition-manifest.md), [ADR-0011](0011-consistency-upgrade-and-recovery.md)

Releases require local P0 evidence and an explicit list of unresolved external
gates. Real PostgreSQL, cache failover, deployment drills, credentials,
penetration testing, and product/legal decisions cannot be satisfied by local
mocks and therefore remain unchecked until evidence is attached.
