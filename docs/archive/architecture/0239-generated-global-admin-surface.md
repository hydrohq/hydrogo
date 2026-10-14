# ADR-0239: Generated globals use a singleton admin surface

- **Status:** Accepted
- **Date:** 2026-09-22
- **Scope:** Generated Admin Studio templates for `type: global`
- **Related:** ADR-0224, ADR-0238

## Decision

Generated admin list templates for global resources must not render collection
tables, pagination, search, or bulk selection. They render a bounded singleton
notice; the generated form remains the editing surface and the transport keeps
the stable `global` ID contract.

This separation prevents the UI from generating erroneous usage patterns as if
a collection route existed. Authorization, save policy, audit, and publication
semantics remain Edition/module-owned; the Core only shapes the safe generated surface.
