# ADR-0048: Bounded test command for the developer workflow

- **Status:** Accepted
- **Date:** 2026-09-14
- **Related decisions:** [ADR-0046](0046-offline-configuration-validation.md), [ADR-0010](0010-sdk-lifecycle-and-edition-manifest.md)

## Context

Developers repeatedly need the same safe test invocation while working in a
generated edition or the Core repository. Copy-pasted shell commands are easy
to run with unbounded duration or accidental shell interpolation.

## Decision

Expose `hydro test` with explicit package, timeout, count, and race flags. The
command invokes `go test` directly through `exec.CommandContext`, streams the
child output, and terminates on the configured deadline. It does not execute a
shell or accept arbitrary command fragments.

## Consequences

- The local feedback loop is discoverable and consistent across platforms.
- Timeout and process boundaries prevent a hung test from consuming an
  unbounded developer/CI session.
- Full verification remains the authoritative release check; this command is a
  convenience wrapper, not a production certification.
