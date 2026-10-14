# ADR-0058: Local `hydro dev` entrypoint

- **Status:** Accepted for HydroGo Core checkouts; scoped by [ADR-0337](0337-hydro-dev-client-edition-entrypoint.md) for scaffolded Client Edition projects
- **Date:** 2026-09-14
- **Related decisions:** [ADR-0045](0045-task-oriented-module-cli.md), [ADR-0337](0337-hydro-dev-client-edition-entrypoint.md)

`hydro dev` delegates to `go run ./tools/devserver` using direct process
arguments and inherits its bounded rebuild, visible error, and browser reload
behavior. The devserver remains local-only; production `serve` is unchanged.

This describes the behavior when `hydro dev` runs inside a HydroGo Core
checkout (detected by `go.mod` + `tools/devserver/`), which is unchanged.
A project scaffolded by `hydro new`/`hydro generate scaffold` has no Go
source tree to `go run`; see [ADR-0337](0337-hydro-dev-client-edition-entrypoint.md)
for the separate entrypoint `hydro dev` uses in that case.
