# Strict Modular Monolith Implementation Plan

This plan establishes strict, verifiable contracts between Core, modules, and external
Client Editions without converting HydroGo into a physical microservices system, allowing
boundaries to transition to separate process boundaries if needed.

## Objectives

- Core provides only shared infrastructure and versioned contracts.
- Modules manage their own domain, service, repository, route, migration, and worker lifecycle.
- Modules never access another module's repository or SQL tables directly.
- An inactive module creates no routes, migrations, assets, workers, or mandatory resources.
- Client Editions cannot access internal Core packages.
- Modules such as Commerce or Search maintain explicit capability and event boundaries that can be factored out into independent processes in the future.

## Phases

1. **Architectural contracts:** ADRs, dependency matrix, import allowlists, and forbidden dependency guards.
2. **Core boundaries:** Consolidate shared errors, pagination, authorization, transactions, audit trails, and provider interfaces into unified Core contracts.
3. **Lifecycle and registry:** Module manifests, centralized registry, dependency and capability tracking; fail-closed behavior for inactive modules.
4. **Capability and event access:** Migrate inter-module concrete service/repository/SQL calls to small interfaces and versioned events.
5. **Migration ownership:** Attach module, dependency, profile ownership, and SQLite/PostgreSQL compatibility validations to migration metadata.
6. **Runtime resource isolation:** Enforce worker, queue, cache, database pool, timeout, and media budgets on a per-module basis.
7. **Admin and frontend isolation:** Derive navigation, dashboard, command palette, and public routes exclusively from active module manifests.
8. **API and SDK isolation:** Generate OpenAPI/SDK artifacts from active modules, maintain legacy aliases at the central router, and enforce contract consistency tests.
9. **CI and test hardening:** Enforce import, lifecycle, inactive resource, migration, capability, event, E2E, and blue/green contract test suites.
10. **Process split readiness:** Module-owned health checks, metrics, queue topics, configurations, serialization formats, and external capability adapter contracts.

## Delivery Discipline

Each phase is delivered via narrow, reviewable changes accompanied by corresponding tests and ADR/validation records. A phase must be green before advancing to the next. Validations requiring real customer scale, reverse proxies, external backup/key escrow, provider credentials, or production signing are never claimed as local success; they remain open production gates.

## Progress Ledger (2026-09-13)

- **Phase 1 — Completed:** Boundaries established in ADR-0037; external Client Edition import guards and internal module import guards wired into CI.
- **Phase 4 — Started:** Direct Members->Commerce domain dependency eliminated in favor of the internal/contracts/customerorders capability contract and Commerce adapter (ADR-0038).
- **Phase 9 — Partially Completed:** Module boundary guards and capability projection regression tests integrated into the CI/Go test pipeline. Remaining migration, lifecycle, event, and blue/green checks scheduled for subsequent phases.
