# ADR-0038: Kit-aware generated runtime wiring

## Status

Accepted

## Decision

Generated `RUNTIME_WIRING.md` records selected capability kits and requires an
explicit module adapter/permission binding for each. Generation remains
deterministic and does not activate optional capabilities implicitly.

## Compatibility and capacity

This is documentation output only; no migration or runtime dependency changes.
Omitted kits preserve the inactive-module invariant and Compact profile budget.
