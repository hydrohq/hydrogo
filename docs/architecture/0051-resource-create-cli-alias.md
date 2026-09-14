# ADR-0051: Resource-oriented CRUD scaffold command

- **Status:** Accepted
- **Date:** 2026-09-14
- **Related decisions:** [ADR-0045](0045-task-oriented-module-cli.md), [ADR-0047](0047-module-authoring-scaffold.md)

## Decision

Expose `hydro resource create --spec SPEC --out OUTPUT` as a collection-oriented
alias of the module scaffold command. Both spellings invoke the same
deterministic generator, policy scaffold, and reviewable CRUD contract. The
alias does not add runtime routes or persistence by itself.
