# ADR-0045: Task-oriented module creation CLI

- **Status:** Accepted
- **Date:** 2026-09-14
- **Related decisions:** [ADR-0010](0010-sdk-lifecycle-and-edition-manifest.md), [ADR-0015](0015-versioned-http-api.md)

## Context

HydroGo already exposes the deterministic collection generator as
`hydro generate module`. Developers think in terms of tasks, and the existing
spelling is easy to miss when following the module-authoring workflow.

## Decision

Expose `hydro module create --spec SPEC --out OUTPUT` as the task-oriented
entrypoint. It delegates to the same deny-by-default generator and validation
pipeline as `hydro generate module`; the historical command remains supported
for backwards compatibility. No new runtime dependency or bypass is added.

## Consequences

- Documentation and onboarding can use a discoverable module-authoring command.
- Both spellings produce identical deterministic artifacts and safety checks.
- A future full module scaffold can extend this command without breaking the
  existing generator contract.
