# ADR-0210: Blueprint-driven scaffold boundary

- **Status:** Accepted
- **Date:** 2026-09-21

## Decision

Client Editions may describe a collection with a small YAML/JSON blueprint and
run `hydro generate scaffold --blueprint PATH --out ROOT`. The command maps the
blueprint into the existing versioned generator contract and emits reviewable
artifacts under `generated/<collection>`, plus a human-owned policy starter in
`custom/`.

The authoring contract is published as
`docs/schemas/blueprint.schema.json`; the Go parser remains the final
fail-closed validation authority and rejects unknown fields.

The blueprint layer never executes DDL, registers runtime routes, or invents
authorization. It supplies conservative default read/write permission names,
requires both SQLite and PostgreSQL metadata, and delegates validation,
artifact hashing, immutable migration handling, and compatibility checks to the
existing generator. Business policy and workflow remain in `custom/`.

## Consequences

- A new collection can be bootstrapped without hand-writing repetitive model,
  repository, API, admin metadata, SDK, and contract-test files.
- Generated output remains reviewable and tamper-detectable through the existing
  generator lock.
- Enum values are currently presentation metadata and do not imply a database
  constraint; a future enum migration contract must be explicit.
- The generator does not replace operator approval and `hydro migrate`.
