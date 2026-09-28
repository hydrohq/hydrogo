# ADR-0273: Optional sectoral content modules

- **Status:** Accepted
- **Date:** 2026-09-28
- **Supersedes ADR:** None
- **Replaced by ADR:** None

## Context

Client Editions need reusable, typed content for locations, teams, partners,
announcements, portfolios, certificates, pricing, events, downloads,
testimonials, FAQs, and legal documents. These collections must remain generic
Core capabilities and must not add work to Editions that do not enable them.

## Decision

- Each collection is an independently enabled module with its own manifest,
  migration, repository contract, service, and typed server-rendered Templ view.
- Shared persistence code owns bounded, tenant-scoped CRUD, validation,
  optimistic version checks, and same-transaction audit records. Writes require
  an audit writer; a missing audit store fails closed.
- Collection queries are bounded to 200 records. Payloads are limited to 1 MiB
  and collection fields have module-specific validation.
- Events may expose an iCalendar download only when its module and calendar
  handler are enabled. Downloads update their count atomically with the record.
- The legal module stores versioned content only. It does not invent or certify
  legal advice or compliance text.
- Related design helpers are passive typed components and do not activate a
  module or register runtime resources.

## Database contract

- SQLite and PostgreSQL use the same logical columns and tenant/locale/slug
  uniqueness. Each module has a separate table and expand-only migration.
- PostgreSQL tables enforce tenant row-level security through the established
  `app.tenant_rls_enforced` and `app.current_tenant_id` transaction settings.
- Repositories use placeholders for the selected dialect, parameterized values,
  tenant predicates, and short transactions. Optimistic updates compare the
  stored version and return a conflict when it has changed.
- Migration metadata records checksums, schema epochs, dialect compatibility,
  smoke checks, and rollback notes. Rollback drops only the selected module's
  collection table after its data has been backed up or intentionally retired.
- Contract tests cover SQLite CRUD, isolation, bounded ordering, version
  conflicts, and opt-in migrations. PostgreSQL contract tests run when
  `HYDRO_TEST_POSTGRES_URL` is configured and verify RLS and CRUD behavior.

## Runtime and capacity impact

- An inactive module registers no routes, migrations, assets, or workers and
  creates no tables. No background worker is introduced by these modules.
- Reads are capped at 200 rows; public transports must paginate or use that
  bounded service contract. Payloads are capped at 1 MiB per record.
- The compact profile uses only the existing database pool and request path.
  No cache, dedicated connection pool, or persistent process is required.
- SQLite writes use short transactions. Larger profiles use the existing
  PostgreSQL pool and tenant RLS; connection and backpressure policy is
  inherited from the database runtime.
- Blue/green rollout is additive: apply enabled module migrations before the
  new binary serves writes. Old binaries ignore the new tables. Rollback uses
  the migration metadata and requires retaining a database backup until the
  rollback window expires.

## Validation

- Module tests cover model validation, service and repository behavior, and
  typed rendered output. Events and downloads additionally test their special
  contracts.
- `templ generate` and `go test ./...` validate generated views and the full
  repository test suite.

## References

- [ADR-0022: Consent-based product analytics and manageable legal content](0022-consent-based-product-analytics-and-legal-content.md)
- [ADR-0228: Public bounded resource query contract](0228-public-resource-query-contract.md)
- [ADR-0233: Public tenant scope contract](0233-public-tenant-scope-contract.md)
- [ADR-0237: Edition-extensible tenant RLS verification](0237-tenant-rls-verification-extension.md)
