# HydroGo — Pre-Implementation Architectural Review and Evidence Plan

**Date:** 2026-09-05  
**Scope:** Manifesto, master plan, ADR-0001–0014, and decision backlog.  
**Current State (2026-09-09):** The repository contains a working Go codebase, CLI, SQLite/PostgreSQL adapters, Pages/News modules, SDK contracts, and contract tests. Open risks and empirical environment evidence are tracked in the [roadmap](ROADMAP.md) and [production gates checklist](validation/PRODUCTION-GATES.md).

## Summary

The core trajectory for HydroGo is explicit: dedicated deployments per customer, a modular Go architecture, Client Editions without HydroCore forks, initial certified databases targeting SQLite and PostgreSQL, and infrastructure layers activated strictly upon demand. The primary risk is no longer a lack of architectural ideas, but rather the empirical execution of the plan.

This review records foundational architectural decisions and reconciles identified contradictions. Route/publishing atomicity and restore erasure log gaps noted in follow-up reviews were resolved via ADR-0014; empirical proof remains open. **Documentation represents Phase 0 deliverables.** Subsequent gates consist of bounded technical experiments and the initial vertical slice. Below, we detail which uncertainty gates which delivery rather than expecting every theoretical question to be closed indefinitely.

## Identified Inconsistencies and Current Decisions

| ID | Issue in Previous Plan | Impact | Current Decision |
|---|---|---|---|
| F01 | News as first product, Site Edition proposed elsewhere | Scope and delivery sequencing diverged | Pages is first slice, News is second module; ADR-0009 |
| F02 | Dedicated deployment still marked open in backlog | Re-debating an already accepted decision | ADR-0001 is active; backlog corrected |
| F03 | Broad Module interface masqueraded as a stable SDK | API frozen prematurely before multiple use cases | Boundaries accepted; Go API v0; v1 after two modules/two Editions; ADR-0010 |
| F04 | Root core/ and co-located client/ isolation was ambiguous | Customers could couple to internal core packages | External Go module, internal + CI allowlist; ADR-0010 |
| F05 | Inactive modules inclusion in binary was unspecified | Zero-overhead promise could not be measured | Selected import/embed via Edition build; lifecycle/parked jobs; ADR-0010 |
| F06 | Rejecting all new schema versions outright | Old slot could not restart after expand migrations | Core/module schema and format compatibility matrix; ADR-0011 |
| F07 | Worker evicted only its local L1 cache | Other web instances served stale content | Per-instance invalidation + generation reconciliation; ADR-0011 |
| F08 | Unhandled publication updates during warmup | Green slot served old data after cutover | Cache bypass/revalidation/flush and stale budgets; ADR-0011 |
| F09 | Claiming cache loss never creates outages | Database fallback could overwhelm origin | Bounded fallback and controlled 429/503 responses; ADR-0011 |
| F10 | Claiming idempotency fields absolutely prevent duplicate emails | Obscured external service non-determinism | At-least-once delivery, provider idempotency, reconciliation; ADR-0011 |
| F11 | Unclear how repository/audit/outbox bound to same transaction | Pseudo-atomicity risking multi-connection splits | Shared transaction scope; ADR-0011 |
| F12 | PostgreSQL restore reduced to directory renaming | Recovery procedures were unworkable | Distinct DB restore + controlled connection switch; ADR-0011 |
| F13 | Assumed brief backup lock during large media copies | Write pauses grew as media volume expanded | Budgeted Compact snapshots; snapshot+GC pin in large profiles; ADR-0011 |
| F14 | Single binary could omit historical embedded assets | Cache-hit HTML missing corresponding CSS/JS | Retained asset inventory and GC window; ADR-0011 |
| F15 | Restore could resurrect obsolete sessions and background jobs | Stale access and duplicate external side effects | Session revocation, job/egress quarantine; ADR-0011 |
| F16 | Implicit early coupling to Quill and locale/revision formats | Impeded future editor and storage evolution | Versioned content envelope, distinct revision/locale; ADR-0012 |
| F17 | Treating object storage write and DB commit as one transaction | Risk of orphan, lost, or leaked private media | Pending/ready + quarantine + GC/backup pinning; ADR-0012 |
| F18 | Assuming database-role immutability for SQLite audit logs | Security promise exceeded underlying platform capabilities | Append-only API, profile-appropriate boundaries, external sinks if required; ADR-0013 |
| F19 | S3 media simultaneously planned and deferred/forbidden | Distributed scope was self-contradictory | Local storage for pilot; S3 prerequisite for Distributed; SQLite remote DB distinct topic |
| F20 | Database contract testing mentioned in production | Risk of test mutations against live customer data | Contracts run in CI/isolated copies; production runs safe smoke/readiness; ADR-0011 |

ADR-0008 was superseded by ADR-0010. Modified provisions in ADR-0003 and ADR-0005 are explicitly annotated; their remaining layer/topology decisions remain in force. Contradictions were not closed silently without linking to updated ADRs.

## Separation of Decisions, Experiments, and Deliveries

| Gate | Current Status | Requirements for Closure |
|---|---|---|
| Architectural Foundation | Documented | Sustaining ADR/plan consistency as implementation progresses |
| Transition to Walking Skeleton | Experiments planned | Core results for S01–S04, module path/toolchain pinning; code provides this evidence |
| Initial Pages Slice | Completed | Dual DB contracts, auth/policy, draft/public separation, local media |
| Commercial Pilot | In progress | S05-C, S06–S07, ADR-0014 external erasure log experiment, MFA, restore, measured capacity, runbooks |
| Stable SDK / v1 | Planned | Two modules + two Editions, upgrade matrix, generator/export, and support policy |
| Distributed | Architectural Target | Multi-instance web/worker/L2/object store failure, load, and consistency drills |

The deliverable for "being confident before coding" is not a speculative guarantee: it consists of testable boundaries and implementation details that can be adapted if an experiment fails. Go interface signatures, ORMs, and view slot mechanisms are not considered stable APIs without empirical evidence.

## Targeted Technical Experiments

Each experiment is conducted with the smallest isolated fixture; it is not automatically promoted to production code. Evaluation reports include code revisions, environment/versions, dataset used, expected vs. observed behavior, failure modes, and the selected alternative. The following were the initial technical experiments:

| ID | Experiment / Concrete Success Criterion | Alternative on Failure | Owner Role / Gate |
|---|---|---|---|
| S01 | External Client Edition cannot import Core internal packages; compiles via allowed extensions; unselected modules do not enter routes/embed/workers | Narrow package graph and build composition | SDK / skeleton |
| S02 | Same service writes content+audit+outbox across both DBs with injected mid-transaction failure; all 3 rollback. Single winner on editor version race | Correct transaction scope and repository contracts; re-evaluate driver choice | DB / first slice |
| S03 | Two small Templ themes compile against identical view models; Quill/HTMX login-save-upload flow operates under strict CSP | Shrink registry; replace incompatible client component; preserve format reader | UI / first slice |
| S04 | SQLite and PostgreSQL driver/migration fixtures run on Windows dev and Linux amd64 release; backup/encoder tools discoverable | Narrow support matrix or swap driver/tools; record decision | Build/DB / skeleton |
| S05-C | Compact profile Blue/Green publication switch, dropped invalidations, and late stale refills; no data leaks, bounded stale budget and DB load. Verify bypass/revalidation during switch | Bypass mutable cache or correct local coordination | Cache / pilot |
| S05-D | Multi-web+worker with L2 fan-out, reconnect/generation reconciliation, lost invalidation, and bounded DB fallback | Bypass mutable cache; correct distributed coordination | Cache / L2 profile; Distributed prerequisite |
| S06 | Old binary restarts after online expand migrations across each DB; reads after new writes on rollback; prior assets operational | Reclassify change as requiring maintenance window or partition into multi-release steps | Release / pilot |
| S07 | Restore DB+media snapshot to clean target; hashes/references match, old sessions revoked, external jobs quarantined | Correct backup/restore flow; withhold RPO/RTO claims | Operations / pilot |
| S08 | Pages and News + two Client Editions run without Core forks; generator yields deterministic zero-diff output and preserves custom hashes | Shrink general interface; narrow generator scope | SDK / v1 |
| S09 | SQLite export → PostgreSQL import round-trip preserves logical identity/revision/locale/relations; secrets/sessions excluded | Correct format/spec; fail-closed on unknown modules | Data / v1 |

Follow-up notes: SEO/redirects belong to the pilot scope; the initial Pages slice covers route/publish atomicity. ADR-0014 external erasure logging and detailed task sequencing are tracked in [Roadmap](ROADMAP.md) Phase R3.

Specific DB/Go/OS/library versions were not arbitrarily invented here. S04 results pin the support matrix; S03 results pin UI dependencies. The Go module namespace is established before the initial `go.mod` using customer-controlled repositories; brand names do not impose technical import constraints.

## Quantifiable Metrics Prior to Initial Pilot

- RPO, RTO, acceptable write pauses, and rollback retention windows.
- Total `max_stale` per content type, publication propagation alerts, and CDN/browser cache policies.
- HTTP body/in-flight/deadline, DB pool, queue, worker, image transform, and password hashing concurrency limits.
- Session idle/absolute timeouts, reset/invite/preview/MFA rate-limit and attempt policies.
- Backup, log/audit, form inbox, revision, and media retention/GC thresholds.
- Acceptable p95/p99 latency and error rates; benchmarks specifying warm/cold cache, write ratios, dataset volume, and hardware.

These numbers are never asserted via speculative marketing claims. They are measured using pilot fixtures and incorporated into config defaults and support matrices. Go memory limits alone are not assumed to bound total process native/cgo/OS memory; RSS and peak memory are measured empirically.

## Proving Development Velocity

Reference scenario: Adding a services collection to an existing Site Edition, defining five fields and one relationship, writing custom validation, binding a list/detail view, and delivering to staging.

Measure the same acceptance criteria using current legacy methods vs. HydroGo. Separately record scaffolding, training, repeat implementation, bug fixing, security/testing, and deployment durations. Repeat across different engineers and a second customer. The 1–2 hour velocity target applies specifically to this standardized workflow; it is not an arbitrary blanket guarantee for bespoke workflows or custom designs.

CLI subcommands may evolve; the developer journey remains constant: Initialize Edition → Select off-the-shelf module → Define standard content → Add custom behavior → Preview locally → Validate → Build/Deploy → Upgrade/Restore as needed. Success is measured by total safe delivery duration, not the quantity of CLI commands.

## Remaining Commercial Decisions

The architecture supports these options; definitive commercial policies must be selected separately:

- Recommendation: Managed service as primary tier, self-hosted as secondary option; license expiration does not disable public sites, but governs updates and support entitlements. Offline operational constraints and enforcement models are not yet accepted product commitments.
- Source code delivery, redistribution rights, dependency licensing, and trademark boundaries must be clarified. "Control remains with us" does not mean intellectual property of third-party dependencies transfers to us.
- The same-major SDK policy is accepted; the number of supported majors, LTS deprecation timelines, and critical security SLAs must be determined prior to commercial launch.
- Trademark and domain availability for HydroGo/HydroCore were not evaluated in this review; distinct research is required prior to marketing materials.

Pricing and legal licensing terms were not decided in this review; the checklist here serves as input to commercial decision-makers.

## Scope of Verification in this Review

Work performed: Document cross-referencing, recording contradictions, drafting new ADRs, linking superseded ADR states, aligning plans and backlogs, and verifying local Markdown links. No Go/DB/operational/security test outcomes are asserted here. Leaving test gates open represents accurate status accounting rather than obscuring issues.

Primary sources for technical mechanisms are cited in the References section of respective ADRs: Go internal boundaries; SQLite concurrency; PostgreSQL DDL; transactional outbox; OWASP session/reset/SSRF guidelines. References do not constitute proof of HydroGo implementation.
