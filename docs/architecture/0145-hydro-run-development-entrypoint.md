# ADR-0145: `hydro run` Development Entrypoint

- Status: Accepted
- Date: 2026-09-18

## Context

HydroGo's live development server was previously accessible only via `hydro dev`, while the README previously referenced the internal tool command `go run ./tools/devserver`. This caused confusion between the supported public CLI surface and internal implementation details.

## Decision

`hydro run` is designated as the canonical user command for local live development. To prevent breaking existing workflows, `hydro dev` is retained as a backwards-compatible alias with identical flags and options. Both paths invoke the unified `devCommand` implementation; the internal `tools/devserver` path is treated as an internal detail rather than public documentation.

## Consequences

- Documentation and user-facing CLI workflows promote `hydro run`.
- Existing scripts and muscle memory using `hydro dev` continue to function identically.
- Server lifecycle and configuration parsing logic remain centralized in a single implementation path.
