# R115 — Generated bounded workflow contract

Date: 2026-09-22

## Scope

Blueprint resources may declare a bounded state graph. The generator validates
state identifiers, duplicate states/transitions, target membership, and the
32-state/transition budgets, then emits deterministic `workflow.json`.

## Evidence

- `go test ./internal/generator -count=1` — PASS.
- Article and Project generated Edition tests pass after regeneration with the
  new required artifact and lock checks.
- Valid Blueprint mapping and generated `enabled/states/transitions` metadata
  are covered by loader tests.
- Unknown transition targets are rejected before artifact creation.
- ADR index drift and formatting checks pass.

## Boundary result

The workflow graph is a versioned, provider-neutral input for Admin Studio and
Edition services. It does not perform persistence, authorization, scheduling,
or side effects; those remain transaction/outbox-owned module behavior.
