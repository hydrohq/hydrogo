# ADR-0287: Dynamic collections on static, certified storage

- **Status:** Proposed
- **Date:** 2026-09-28
- **Related decisions:** [ADR-0020](0020-headless-first-admin-studio-and-frontend-templates.md), [ADR-0069](0069-optional-module-lifecycle-contracts.md), [ADR-0080](0080-profile-pool-transaction-contract.md), [ADR-0134](0134-site-module-business-capability-boundary.md)

## Context

Issue #29 asks editors to define collections and fields in Admin Studio. The
repository's migration and generator contracts prohibit executing DDL while a
schema spec is loaded. This capability must remain optional and portable across
SQLite, PostgreSQL, and the supported MySQL adapter.

## Decision

- Collection schemas are validated JSON contracts in the optional `collections`
  module. Creating a collection writes metadata only; it never executes DDL or
  creates a per-collection table.
- A module migration creates a fixed set of tenant-scoped tables for schemas,
  records, unique-value hashes, and relation edges. Data is kept in bounded JSON
  records. All field names remain data and are never interpolated into SQL.
- Text, rich text, number, boolean, date-time, single/multi select, media,
  single relation, and multi relation fields use one versioned field contract.
  Validation is performed before persistence. Unique values use a static
  database uniqueness constraint; relation edges use static foreign keys.
- The versioned `/api/v1/collections/{collection}` surface and Admin Studio
  CRUD use the same application service. Authorization, tenant scope, and
  record validation are enforced at that boundary.
- The module contributes no migration, route, assets, or mandatory services
  when inactive. In Compact profile it enforces fixed limits for collections,
  fields, records per collection, list page size, and serialized record size.
- Removing or changing a schema is an explicit application operation. Field
  rename/type changes require an exportable schema revision and compatibility
  review; they do not silently rewrite records.

## SQLite/PostgreSQL and deployment contract

The initial migration is additive and has no rollback DDL. SQLite and
PostgreSQL migrations create the same logical keys and constraints. MySQL gets
the same constraints with bounded `VARCHAR` key columns. Contract tests must
exercise schema creation, tenant isolation, record CRUD, unique collisions,
relation existence/deletion protection, and migration filtering on SQLite and
PostgreSQL. Blue/Green compatibility is preserved by keeping the storage
version and JSON fields additive; old binaries ignore fields unknown to their
schema reader and writes remain gated by the schema version.

The tables grow with user data, so deployments must include them in existing
database pool and backup capacity planning. There is no worker or cache. List
queries are page-bounded and indexes begin with tenant and collection IDs.
Compact deployments cap collection definitions, fields, records, and payload
size to keep validation and writes bounded.

## Alternatives

- Per-collection DDL at runtime: rejected because it bypasses certified
  migration, rollback, and Blue/Green compatibility workflows.
- Store all records in one unindexed global blob: rejected because it makes
  tenant filtering, unique fields, and relations expensive or unsafe.
- Generate a migration for every collection: remains available to Editions
  that need physical tables, but is not the zero-code runtime path.

## Acceptance gates

This proposal becomes accepted after the SQLite/PostgreSQL contract suite,
generated Templ Admin Studio screens, API authorization checks, relation and
unique-value concurrency tests, and module-inactive migration/route checks pass.
