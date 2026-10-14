# ADR-0202: Core/Client Edition boundary and Construction pilot

- **Status:** Accepted
- **Date:** 2026-09-21
- **Related:** [ADR-0010](0010-sdk-lifecycle-and-edition-manifest.md), [ADR-0140](0140-tenant-isolation-scope.md), [ADR-0147](0147-single-tenant-production-deployment-and-bff-model.md), [ADR-0149](0149-shared-multi-tenant-saas-evolution.md), [ADR-0195](0195-admin-extension-channel-binding.md), [ADR-0200](0200-reviewed-collection-designer.md)

## Context

HydroGo needs to support a Construction product without turning project,
contractor, payment, document, or approval details into compulsory Core
behavior. The first commercial Construction deployment also needs a safer
operational boundary than the proposed shared-database SaaS topology.

## Decision

1. HydroGo Core owns only reusable platform capabilities: identity and session
   infrastructure, authorization primitives, audit, migrations, media
   abstraction, outbox/webhook delivery, API contracts, release controls, and
   capacity profiles.
2. Construction is a separately versioned Client Edition. It may contribute
   selected, reviewed module contracts, but it must not import Core `internal/`
   packages or require a Core fork.
3. Construction domain concepts, including project hierarchy, site reports,
   contractor access, document review, approval workflow, progress tracking,
   and any future commercial ledger, remain in the Client Edition until they
   are independently proven reusable in two products.
4. The first live Construction pilot uses the per-customer isolated topology:
   one HydroGo instance, PostgreSQL database, and S3-compatible bucket or
   prefix per customer. Shared-database tenancy remains disabled until every
   ADR-0149 release gate has production evidence.
5. The pilot begins with organization/project hierarchy, scoped roles,
   document revisions, daily field reports, issue tracking, and explicit
   approval transitions. It excludes accounting, payroll, e-invoicing,
   procurement, payment processing, arbitrary runtime schema changes, and
   shared-database SaaS activation.

## Alternatives

### Put Construction directly in Core

Rejected because it would impose vertical routes, migrations, workers, and
business vocabulary on unrelated Editions, violating inactive-module and
two-real-use-case rules.

### Start with shared-database multi-tenancy

Rejected for the pilot because ADR-0149 is proposed and its RLS, IDOR,
noisy-neighbor, recovery, and operational evidence gates are not yet closed.

### Use runtime Go plugins

Rejected because an untrusted plugin is not a portable or meaningful security
boundary and conflicts with ADR-0010 and the signed external-surface model.

## Consequences

### Benefits

- The Construction product can evolve without contaminating Core.
- Customer data, blast radius, backup/restore, and incident handling are
  physically isolated in the first commercial stage.
- Reusable platform work remains versioned and testable across Editions.

### Costs and risks

- The pilot needs repeatable per-customer infrastructure automation.
- Shared SaaS unit economics are deliberately delayed.
- Domain modules must supply their own authorization and workflow tests rather
  than inheriting an implied generic ERP model.

## Scale impact

- **Compact:** No Construction service or asset is included unless the Edition
  selects it.
- **Standard:** The supported pilot profile uses PostgreSQL, Redis/Valkey when
  configured, and S3-compatible media.
- **Distributed:** Not a Construction pilot claim; requires the existing
  distributed-profile certification gates and measured backpressure.

## Database impact

- **SQLite:** Supports local development and contract testing only for the
  pilot; it is not the intended multi-user production database.
- **PostgreSQL:** Required for the live pilot; Construction migrations must be
  dialect-reviewed and follow expand-migrate-contract release discipline.
- **Future adapters:** Need explicit Construction module certification before
  being supported.

## Cache impact

Construction cache keys are edition/module scoped. Any future tenant-scoped
cache use must call the Core scope API; project IDs are never substituted for a
tenant authorization boundary. Cache entries need bounded TTL, invalidation on
approved mutations, stampede protection, and a safe database fallback.

## Deployment and compatibility

Construction modules are selected during the release build. HTTP startup does
not execute DDL. A release must include migrations, compatibility metadata,
old/new job payload treatment, retained assets, and rollback behavior. The
pilot may not be promoted to shared tenancy through configuration alone.

## Validation

- An external Construction Edition builds without Core source copying or
  `internal/` imports.
- Unselected Construction modules have no routes, migrations, workers, or
  assets in a Site Edition build.
- Each Construction workflow has SQLite/PostgreSQL transaction, authorization,
  audit, and optimistic-concurrency contract tests.
- A live pilot passes the production restore, blue/green, capacity, and
  independent-security gates before a production-readiness claim.
