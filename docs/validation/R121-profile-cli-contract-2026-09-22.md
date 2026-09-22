# R121 — Capacity profile CLI contract

Date: 2026-09-22

## Evidence

- `hydro profile explain --profile compact` reports the Compact DB/cache/
  worker/queue defaults and the declared capability manifest.
- `hydro profile check --profile compact --json` emits valid machine-readable
  JSON; standard and distributed checks also return success.
- Unit tests cover Compact exclusions for distributed cache/OTLP, distributed
  capability visibility, malformed command/profile rejection, and the JSON
  result envelope.
- `go test ./cmd/hydro ./internal/capacity -count=1` — PASS.

## Boundary

This verifies the CLI and declared capacity contract only. The current
capability manifest is metadata, not an Edition-resolved dependency/link graph
or proof that every inactive service, worker, route, migration and asset is
absent. P0 remains open for Edition-aware composition and measured artifact/
runtime budgets.
