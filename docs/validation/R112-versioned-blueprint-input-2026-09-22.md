# R112 — Versioned Blueprint input boundary

Date: 2026-09-22

## Scope

Blueprint YAML/JSON input now has an explicit `version` boundary. Omitted
versions remain compatible as `v0`; only `v0` is currently accepted. Unknown
future versions fail before `Spec` creation or generated output.

## Evidence

- `go test ./internal/generator -run 'TestLoadBlueprint(RejectsUnsupportedVersion|AcceptsExplicitV0)' -count=1` — PASS.
- The strict YAML/JSON decoders reject unknown fields as before.
- `docs/schemas/blueprint.schema.json` allows only `v0` and defaults omitted
  input to `v0`.
- ADR index drift check covers the new decision record.

## Boundary result

Schema evolution is now deliberate and reviewable. Existing Blueprints remain
valid, while a future schema cannot silently change generated migrations,
transport, or Admin Studio artifacts.
