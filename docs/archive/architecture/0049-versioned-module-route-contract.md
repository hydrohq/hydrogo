# ADR-0049: Versioned module route contract

- **Status:** Accepted
- **Date:** 2026-09-14
- **Related decisions:** [ADR-0010](0010-sdk-lifecycle-and-edition-manifest.md), [ADR-0037](0037-strict-modular-monolith-boundaries.md), [ADR-0047](0047-module-authoring-scaffold.md)

## Context

Client Editions need a stable import boundary for module authoring. Importing
the root `extension` package couples editions to the Core package layout and
makes future contract evolution harder to communicate.

## Decision

Publish the module route and module interfaces under `extension/v1`. The first
version is a source-compatible alias of the existing root contract, so current
modules continue to compile while new modules can depend on the versioned
namespace. `PublicRoute` is the only convenience constructor; protected routes
remain explicit declarations with `Public: false`.

This is an SDK boundary, not a second runtime registry. Core keeps ownership of
registration and lifecycle behavior, and no internal implementation types are
re-exported.

## Consequences

- Editions have a discoverable, versioned import path for module routes.
- Existing modules do not require a flag day migration.
- Future breaking contract changes can be introduced as `extension/v2` without
  leaking SQL, cache, or transport internals into the SDK.
- The broader capability/version matrix and lifecycle API remain separate work.
