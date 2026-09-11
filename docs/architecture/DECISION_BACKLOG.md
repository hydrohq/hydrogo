# HydroGo Architecture Decision Backlog

**Update:** 2026-09-09. Decision status and implementation status are tracked separately. Even though some decisions have implementations and contract proofs, actual environment gates may still be open; current status is kept in the [roadmap](../ROADMAP.md) and validation reports. Proof plan is in the [pre-implementation review](../PRE_IMPLEMENTATION_REVIEW.md), active ADR index is in [README](README.md).

## Closed fundamental decisions

| Topic | Current answer | Decision source |
|---|---|---|
| Initial product | Site Edition; Pages first slice, News second validation module | [ADR-0009](0009-site-edition-and-delivery-scope.md) |
| Tenancy | Separate deployment/data per customer; first release is not multi-tenant | [ADR-0001](0001-product-and-deployment-model.md); already accepted |
| Database | Initial certification SQLite/PostgreSQL; subsequent engines via test kit | [ADR-0006](0006-extensible-database-adapters.md) |
| Schema/custom code | Standard CRUD from build spec; complex workflow is module; no runtime automated DDL | [ADR-0009](0009-site-edition-and-delivery-scope.md) |
| SDK/Edition | External Go module, narrow extension, experimental v0, v1 after two modules/two Editions | [ADR-0010](0010-sdk-lifecycle-and-edition-manifest.md) |
| Generated/custom | Deterministic diff; custom and published migration preserved | [ADR-0010](0010-sdk-lifecycle-and-edition-manifest.md) |
| Module lifecycle | Selected build, controlled shutdown, jobs parked, data not automatically deleted | [ADR-0010](0010-sdk-lifecycle-and-edition-manifest.md) |
| Release manifest | Edition intent, dependency lock and build proof are separate; secrets external | [ADR-0010](0010-sdk-lifecycle-and-edition-manifest.md) |
| Migration/rollback | Per-dialect metadata, schema and payload compatibility; binary rollback separate, restore separate | [ADR-0011](0011-consistency-upgrade-and-recovery.md) |
| Cache | L0–L3; instance fan-out/generation, stale budget, bounded failure fallback | [ADR-0003](0003-cache-architecture.md) + [ADR-0011](0011-consistency-upgrade-and-recovery.md) |
| Queue/outbox | DB-backed at least once, transaction scope, lease/token, idempotency/quarantine | [ADR-0011](0011-consistency-upgrade-and-recovery.md) |
| Content/locale | Typed relational model, separate revision/locale, versioned rich-text | [ADR-0012](0012-content-media-and-portability.md) |
| Media | Local first; S3 relevant profile gate; immutable/pending-ready/private-public | [ADR-0012](0012-content-media-and-portability.md) |
| Export/search | Logical export; search is reproducible and authority-filtered projection | [ADR-0012](0012-content-media-and-portability.md) |
| Auth/security | Local auth, DB session, pilot MFA, registration/domain authority; external identity later | [ADR-0013](0013-identity-and-security-boundaries.md) |

## Open tasks and deliveries they block

P0: first implementation detail; P1: pilot/v1 delivery; P2: expansion by demand. The open tasks in this table do not block the first code.

| Priority | Topic | Proposal / required for decision | Gate | Status |
|---|---|---|---|---|
| P0 | Go module namespace | github.com/ardaozyildirim/hydrogo; created in go.mod/toolchain R1 experiment | First go.mod | Implemented; verified with go.mod, CI and import guard |
| P0 | Driver/toolchain/migration tool | Two DBs + Windows dev/Linux amd64 fixture; version/build/license evaluation | Skeleton S02/S04 | Core/CI and migration contracts implemented; provider certification still open |
| P0 | SDK signatures/transaction executor | Small registration and transaction scope; wide interface not to be frozen | S01/S02, stability S08 | Implemented; extension/v1 and transaction contracts verified |
| P0 | UI dependencies/view slot | Templ + candidate Quill/HTMX/Alpine CSP; verification with two themes | S03 | Implemented; versioned asset, CSP and theme contracts verified |
| P0 | Slug/email canonicalization | Normalize helper fields; Unicode/case/locale fixtures; data portability | First persistent schema | Implemented; common NFC/case/URL-path canonicalization and regression fixtures available; real data migration/provider proof still open |
| P1 | Numerical resource/security/cache defaults | Measured limits and total max_stale; no silent infinite defaults | Pilot/profile | Typed profile budgets, bounded cache/queue/body/upload limits implemented; real capacity measurement pending |
| P1 | Restore and RPO/RTO | Real restore for two DBs; media/key and egress quarantine | Pilot S07 | Pending proof |
| P1 | Restore deletion history | ADR-0014 external durable intent log, current checkpoint, retention and crash experiment | Pilot R3 | Pending provider/protocol proof |
| P1 | DB/OS/provider supported versions | Real CI matrix, backup/encoder requirements and certification | Relevant release | Open |
| P1 | License/offline behavior | Proposal: public site does not shut down when expired; support/update right is separate | Pre-sales | Commercial decision open |
| P1 | Source code/redistribution | Customer delivery and dependency licenses should be clearly defined | Pre-sales | Commercial decision open |
| P1 | LTS/deprecation/SLA | Same-major SDK policy accepted; concrete durations/support window must be selected | Stable v1/sales | Commercial decision open |
| P1 | Brand/domain | Separate suitability research for HydroGo/HydroCore | External marketing | Not verified |
| P1 | Portable export/import | Two DB round-trips; format/module/secret rules | v1 S09 | Allowlist, SQLite/PostgreSQL round-trip and CLI implemented; real customer hardware benchmark pending |
| P2 | Advanced search/object storage | Interface principles accepted; provider semantics and load test required | Relevant feature/Distributed | Pending implementation |
| P2 | OIDC/SAML/custom mobile API | Separate flow, threat model, token/revoke and adapter ADR | Before feature opens | Postponed |
| P2 | Fleet | CI + version inventory in first customers; dashboard upon proven need | Operations demand | Postponed |
| P2 | Commerce | Separate edition and money/stock/ledger/transaction decisions | New product | Postponed |
| P2 | Multi-region/multi-tenant | Out of initial product scope; requires real demand and new decision | New topology | Postponed |

## Development velocity target

Standard customer work involves measuring edition startup, theme, collection, custom validation, test, build/deploy and bug fix times. The 1–2 hours target can only be a product claim after repeated pilot measurement in defined work. No time promises are made as if there is a ready-made generator or SDK.
