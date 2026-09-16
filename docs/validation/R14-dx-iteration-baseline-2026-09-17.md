# R14 developer iteration baseline — 2026-09-17

## Scope

This is a repeatable local baseline for the first-project and first-resource
workflow. It measures the current HydroGo generator path; it is not a claim of
Payload parity or a comparison between two developers.

## Environment

- Windows amd64, Go 1.27.0
- Revision: `581a60c6080c9a51109222e60eeb1c2470a358f4`
- Generator input: `examples/module-starter/collection.yaml`

## Result

The acceptance steps completed in **2.691 seconds** (threshold: 15 minutes):

- first project scaffold: 1.069 s
- generated route manifest check: 0.050 s
- first resource generation: 0.807 s
- generated compatibility check: 0.765 s

Supporting checks also passed: targeted UI/contract tests (3.031 s) and full
compile (13.454 s). The machine-readable result is kept in
`R14-dx-iteration-baseline-2026-09-17.json`.

## Limitations and next evidence

This baseline does not measure a second developer, a clean cold-cache machine,
customer-sized projects, browser authoring time, PostgreSQL, or production
deployment. The roadmap item remains open until the same standard job is
measured on two independent developer/customer environments and the methodology
and variance are recorded.
