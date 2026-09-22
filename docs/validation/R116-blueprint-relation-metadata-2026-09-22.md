# R116 — Blueprint bounded relation metadata

Date: 2026-09-22

## Scope

Blueprint relation targets and cardinalities now map into the generated Spec.
Admin forms receive relation widget metadata and every resource emits a
deterministic `relations.json` contract. Many-valued references accept a
bounded `max_items` (default 50, hard maximum 100).

## Evidence

- `go test ./internal/generator -count=1` — PASS.
- Tests cover one-valued required relation mapping, many-valued item budgets,
  generated relation manifest, admin selector metadata, and out-of-range reject.
- Generated compatibility requires `relations.json` and validates its lock
  checksum.
- JSON Schema declares only target, cardinality and bounded `max_items`.
- ADR index drift and formatting checks pass.

## Boundary result

This closes Blueprint-to-contract metadata loss. It does not claim generated
many-to-many persistence or eager loading; normalized relation storage,
authorization, tenant scope, cascade policy, and provider query plans remain
Edition-level acceptance criteria.
