# ADR-0037: Strict modular monolith boundaries

- **Status:** Accepted
- **Date:** 2026-09-13
- **Related decisions:** [ADR-0001](0001-product-and-deployment-model.md), [ADR-0010](0010-sdk-lifecycle-and-edition-manifest.md), [ADR-0024](0024-optional-redis-rate-limiting-and-queue-boundary.md), [ADR-0028](0028-core-isolation-and-licensing.md)

## Decision

HydroGo remains a modular monolith for the current product phase. Core
infrastructure, versioned module contracts, and Client Editions are separated
by enforced import, lifecycle, capability, migration, resource, and test
boundaries. Modules may use Core contracts but may not directly use another
module's repository, SQL tables, workers, or concrete services.

Inactive modules must not register routes, migrations, assets, workers, or
mandatory resource overhead. Optional cross-module behavior is represented by a
small capability interface or versioned event. The composition root is the only
place that assembles active modules.

The boundaries must be designed so a future module such as Commerce or Search
can move to a separate process without changing domain contracts, but physical
process separation is not required until independent scaling, trust, failure,
or regulatory requirements justify it.

## Consequences

This preserves Compact-profile simplicity and single-transaction behavior while
preventing accidental coupling and inactive-module overhead. It requires
additional registry, import-guard, migration, and contract tests. Shared Core UI
and infrastructure remain allowed; module-specific behavior must stay inside
the module or its published contract.

Registry names are canonicalized (trimmed and lowercased) before duplicate
detection so module composition cannot diverge because of presentation-only
spelling differences.
