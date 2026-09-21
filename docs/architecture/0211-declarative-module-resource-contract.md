# ADR-0211: Declarative module resource contract

- **Status:** Accepted
- **Date:** 2026-09-21

## Decision

Optional modules may expose a `modules.Manifest` through the versioned module
composition boundary. The manifest declares dependencies, migration IDs,
assets, and worker IDs. The registry exposes deterministic manifests and an
explicit `RegisterIfEnabled`/`RegisterEnabled` path.

An inactive module is not registered and therefore contributes no routes or
resource metadata. The registry validates composition and reports declarations;
it never executes migrations, loads assets, or starts workers. Those actions
remain owned by the Edition composition root and operator lifecycle commands.
Legacy modules implementing only the existing route contract remain supported
while Editions migrate to the declarative interface.

## Consequences

- Inactive-module zero-overhead behavior is testable at the registry boundary.
- Edition tooling can inspect dependencies and resource ownership before startup.
- Migration and worker activation remain explicit, preserving no-runtime-DDL and
  bounded Compact-profile behavior.

