# ADR-0099: Deterministic capability-kit dependency resolution

**Status:** Accepted  
**Date:** 2026-09-16

## Context

Selecting a capability kit manually can omit a required companion capability,
which slows prototypes and produces runtime surprises. Dependency resolution
must not load implementations or add inactive-module footprint.

## Decision

Admin Studio exposes `ResolveKits`, which validates declared kits, expands the
small allowlisted dependency metadata, and returns a sorted set. The current
contract declares Commerce → Forms. Kit implementations, routes, migrations,
workers, and resource budgets remain module-owned and opt-in.

## Consequences

Generated manifests can use one deterministic kit set. Unknown or duplicate
kits fail closed. Adding a dependency requires an explicit contract/test
change; no arbitrary runtime package loading is introduced.

## Verification

`go test ./internal/adminstudio -count=1` covers dependency expansion and
deterministic ordering.
