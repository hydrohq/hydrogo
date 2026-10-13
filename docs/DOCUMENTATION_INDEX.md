# HydroGo Documentation Index

This document maps where current decisions, specifications, and validation records live.
For a capability to be considered complete, not only the code, but the corresponding ADR,
test evidence, and operational notes must be verified together.

## Where to Start

| Need | Primary Document |
|---|---|
| Philosophy and product boundaries | [HYDROGO_PHILOSOPHY.md](HYDROGO_PHILOSOPHY.md) |
| Information hierarchy & cognitive clarity | [INFORMATION_HIERARCHY.md](INFORMATION_HIERARCHY.md) |
| Execution sequence | [implementation_plan.md](implementation_plan.md), [ROADMAP.md](ROADMAP.md) |
| High-level status | [PROJECT_STATUS.md](PROJECT_STATUS.md) |
| Detailed implementation & open gates | [IMPLEMENTATION_STATUS.md](IMPLEMENTATION_STATUS.md) |
| Remaining real-world infrastructure gates | [OPEN_GATES_AND_OPERATOR_ACTIONS.md](OPEN_GATES_AND_OPERATOR_ACTIONS.md), [IMPLEMENTATION_AUDIT_REMAINING.md](IMPLEMENTATION_AUDIT_REMAINING.md) |
| Architecture decision records | [architecture/README.md](architecture/README.md) and related ADRs |
| Project generation and SDK CLI | [PROJECT_GENERATOR.md](PROJECT_GENERATOR.md), [CLI_WORKFLOW.md](CLI_WORKFLOW.md), [HYDRO_SDK_PRIVATE_RUNTIME_DISTRIBUTION.md](HYDRO_SDK_PRIVATE_RUNTIME_DISTRIBUTION.md) |
| SDK and extension architecture | [SDK_CAPABILITY_MATRIX.md](SDK_CAPABILITY_MATRIX.md), [SDK_SUPPORT_POLICY.md](SDK_SUPPORT_POLICY.md) |
| Payload speed & parity execution plan | [PAYLOAD_PARITY_EXECUTION_PLAN.md](PAYLOAD_PARITY_EXECUTION_PLAN.md) |
| Core bloat and Compact profile audit | [CORE_BLOAT_AUDIT.md](CORE_BLOAT_AUDIT.md) |
| Benchmark protocol and measurement | [PAYLOAD_BENCHMARK_PROTOCOL.md](PAYLOAD_BENCHMARK_PROTOCOL.md) |
| HydroGo iteration baseline | [validation/R5-iteration-baseline-2026-09-16.md](validation/R5-iteration-baseline-2026-09-16.md) |
| Public SDK and runtime distribution | [HYDRO_SDK_PRIVATE_RUNTIME_DISTRIBUTION.md](HYDRO_SDK_PRIVATE_RUNTIME_DISTRIBUTION.md), [ADR-0091](architecture/0091-sdk-runtime-artifact-distribution.md) |
| Binary-only distribution packaging | [COMMERCIAL_LICENSING.md](COMMERCIAL_LICENSING.md), [R64 validation](validation/R64-binary-only-customer-package-2026-09-17.md), [ADR-0143](architecture/0143-binary-only-customer-distribution.md) |
| SDK artifact broker API | [HYDRO_SDK_BROKER_API.md](HYDRO_SDK_BROKER_API.md) |
| SDK broker client evidence | [validation/R5-sdk-broker-client-2026-09-16.md](validation/R5-sdk-broker-client-2026-09-16.md) |
| Deployment & operations | [DEPLOYMENT.md](DEPLOYMENT.md), [OPERATIONS_RUNBOOK.md](OPERATIONS_RUNBOOK.md), [RELEASE_CHECKLIST.md](RELEASE_CHECKLIST.md) |
| Security threat model | [security/THREAT_MODEL.md](security/THREAT_MODEL.md), [COMMERCIAL_LICENSING.md](COMMERCIAL_LICENSING.md) |
| Security scanning & reports | [validation/security-check-2026-09-16.md](validation/security-check-2026-09-16.md), [`security-report/`](../security-report/) |
| Local production and capacity evidence | [R65 validation](validation/R65-local-production-and-capacity-2026-09-17.md) |
| Security skill integration | [SECURITY_CHECK_INTEGRATION.md](SECURITY_CHECK_INTEGRATION.md) |
| Frontend and Admin Studio | [FRONTEND_ARCHITECTURE.md](FRONTEND_ARCHITECTURE.md), [HEADLESS_ADMIN_STUDIO_PLAN.md](HEADLESS_ADMIN_STUDIO_PLAN.md), [UI_DESIGN_SYSTEM.md](UI_DESIGN_SYSTEM.md) |

## Document Types

- `docs/architecture/`: Rationale for architectural choices; marked Proposed until formally accepted.
- `docs/validation/`: Executed commands, environment details, expected/observed results, and limitations.
- `docs/*_PLAN.md` or backlog documents: Work to be done; not proof of completion.
- `PROJECT_STATUS.md`: Executive summary status; not a production readiness guarantee.

## Status Vocabulary

- **Implemented / Closed:** Code and certified contract tests/evidence exist.
- **Planned:** Design or task defined; implementation evidence not yet verified.
- **Open gate:** Awaiting real environment verification, external vendor integration, legal review, or release rehearsal.
- **Not supported:** Intentionally out of scope; rationale and alternative documented.

## Required Updates for Architectural Changes

1. Affected or new ADR.
2. Implementation and database contract tests.
3. Migration and compatibility metadata.
4. Dated validation report.
5. Roadmap/status and capacity/deployment impact notes.
6. Information hierarchy verification for all user-facing copy.

If a validation report does not exist, the result is never marked complete; planned work remains open.
