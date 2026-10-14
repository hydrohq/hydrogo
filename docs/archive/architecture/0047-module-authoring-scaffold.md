# ADR-0047: Reviewable module authoring scaffold

- **Status:** Accepted
- **Date:** 2026-09-14
- **Related decisions:** [ADR-0010](0010-sdk-lifecycle-and-edition-manifest.md), [ADR-0045](0045-task-oriented-module-cli.md)

## Context

Fast development requires a predictable starting point, but generated code
must not silently invent authorization or business policy. The collection
generator previously emitted only persistence, transport, migration, and admin
metadata artifacts.

## Decision

`hydro module create` emits a reviewable scaffold containing model, repository,
service, transport, deny-by-default policy interface, fixture/contract tests,
admin manifest, module metadata, and a non-destructive migration draft. The
service is intentionally thin and the policy is an interface: no mutation is
exposed without explicit application code and authorization review.

## Consequences

- A new standard collection has a complete, navigable development surface from
  the first command.
- Determinism, secret-field rejection, symlink protection, and immutable
  migration behavior remain inherited from the existing generator.
- Generated output is not a production feature until its policy, transaction,
  audit, and migration behavior is reviewed and tested.
