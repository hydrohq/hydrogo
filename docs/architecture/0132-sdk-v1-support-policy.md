# ADR-0132: SDK v1 support and deprecation policy

- **Status:** Accepted
- **Date:** 2026-09-16
- **Related decisions:** [ADR-0010](0010-sdk-lifecycle-and-edition-manifest.md), [ADR-0066](0066-reproducible-release-evidence.md)

## Decision

HydroGo v1 uses semantic compatibility rules, 12-month support for the latest
minor line and a 24-month LTS line beginning each calendar year. Deprecations
have a documented replacement and migration path and remain for at least two
minor releases or six months, except where a security fix requires faster
action. Release manifests and contract fixtures are the source of evidence.

This decision deliberately does not select commercial licensing, paid SLA
terms or redistribution rights; those require a separate business/legal
approval and cannot be implied by an SDK version.

## Evidence

The policy is recorded in `docs/SDK_SUPPORT_POLICY.md`. Existing generator,
external-consumer and compatibility checks provide the release evidence
surfaces; official v1 release signing and commercial terms remain open gates.
