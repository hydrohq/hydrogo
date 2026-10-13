# ADR-0244: Generated resources expose bounded workflow metadata

- **Status:** Accepted
- **Date:** 2026-09-22
- **Scope:** Blueprint workflow declaration and generated `workflow.json`
- **Related:** ADR-0122, ADR-0227, ADR-0241

## Decision

A Blueprint may declare a bounded workflow with 1–32 unique lowercase states
and transition targets that reference only declared states. The generator
emits deterministic `workflow.json` metadata for every resource; disabled
resources carry an explicit empty contract.

The contract is descriptive and provider-neutral. It does not execute a state
transition, grant a role permission, schedule publication, write a database
row, or perform a side effect. Editions must validate the transition and
optimistic-concurrency policy inside their service transaction and dispatch
external effects through the durable outbox/lifecycle boundary.

## Consequences

- Resource authors get a single, reviewable state graph that can feed Admin
  Studio and module-owned services.
- Invalid or unbounded graphs fail before migrations or artifacts are written.
- Core remains free of moderation semantics and runtime workers.

## Verification

Loader tests cover valid mapping, deterministic generated metadata, unknown
transition rejection, and the generated artifact compatibility list. The
runtime wiring checklist requires transaction/outbox binding when a workflow
is declared.
