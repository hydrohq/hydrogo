# ADR-0058: Local `hydro dev` entrypoint

- **Status:** Accepted
- **Date:** 2026-09-14
- **Related decisions:** [ADR-0045](0045-task-oriented-module-cli.md)

`hydro dev` delegates to `go run ./tools/devserver` using direct process
arguments and inherits its bounded rebuild, visible error, and browser reload
behavior. The devserver remains local-only; production `serve` is unchanged.
