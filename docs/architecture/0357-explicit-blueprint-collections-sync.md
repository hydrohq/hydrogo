# ADR-0357: Explicit Blueprint sync into Dynamic Collections

- **Status:** Accepted
- **Date:** 2026-10-08
- **Supersedes:** None
- **Superseded by:** None
- **Related decisions:** [ADR-0200](0200-blueprint-spec-designers.md), [ADR-0287](0287-static-storage-dynamic-collections.md), [ADR-0354](0354-blueprint-derived-admin-activation.md), [ADR-0355](0355-aggregate-generated-resource-route-registration.md)

## Context

Blueprint resources currently produce generated Edition models, repositories,
migrations, API contracts, and Admin metadata. A generated Admin registry still
needs Edition-owned route, policy, and migration wiring. ADR-0287 separately
provides bounded schema-driven CRUD through the optional `collections` module.
The two runtimes have different authorization contracts, so silently treating
one as the other would grant access without Edition policy review.

## Decision

- Provide `hydro collections sync` as an explicit operator command that imports
  compatible generated Blueprint schemas into the existing static-storage
  Collections module. It does not activate generated resource routes or change
  the Blueprint-generated Edition runtime.
- The Collections module must already be enabled in Edition config and its
  existing migration must already be applied. Sync executes no DDL, migration,
  route registration, module activation, permission grant, or worker startup.
- Require `--confirm-admin-crud`. The operator must review the Edition's
  generated resource authorization before accepting that existing Admin Studio
  administrators can perform full CRUD through `/admin/collections`. Only
  resources declaring both a `.read` and `.write` permission are eligible;
  this is a guard against obvious read-only resources, not a translation of
  generated permission callbacks.
- Map only field types and constraints that the Collections runtime represents
  faithfully. Reject globals, auth resources, workflow/draft/revision/localized
  resources, unsupported validations and field types. Hidden resources are
  skipped. A tenant Edition imports only tenant-scoped specs, requires a
  configured tenant ID via `--tenant`, and syncs that tenant's schema rows.
- Create missing schemas only. Existing identical schemas are idempotent;
  schema drift, relation cycles, and unresolved relation targets fail without
  overwriting or deleting data. Relation targets are created before dependents.
  If a database error occurs after some inserts, rerunning safely skips those
  schemas and continues.
- The resulting experience is the existing Admin-only Collections UI and its
  CRUD workflow. It does not inherit generated Admin visual metadata such as
  groups/actions or generated Edition role/permission callbacks.

## Alternatives

- Automatically register generated resources in the Edition: already covered
  by ADR-0354/0355 and remains the preferred path when Edition callbacks and
  migrations are wired.
- Make Collections sync implicit during Blueprint generation: rejected because
  generation must not execute database writes, enable modules, or alter runtime
  authorization.
- Generate per-resource tables and routes: rejected for this bounded bridge;
  that would change module, migration, deployment, and authorization contracts.
- Treat generated permissions as Collections policy: rejected because names in
  a Blueprint do not provide the Edition's identity and policy callbacks.

## Consequences

### Benefits

- An operator can expose simple generated content in the existing Admin CRUD
  panel without generating or applying per-resource DDL.
- The import is repeatable and fails closed for schema drift or unsupported
  behavior.

### Costs and risks

- Collections Admin access is not equivalent to generated Edition callbacks.
  Operators must review the consequence of full Admin CRUD before confirming.
- Blueprint list presentation, custom actions, generated API behavior,
  editorial workflow, versioning and localization are not transferred.
- Tenant sync is an explicit per-tenant operation and adds only schema metadata
  inside the selected tenant scope.

## Scale impact

- **Compact:** no runtime overhead when the optional module is inactive. When
  enabled, sync is a one-shot CLI process using existing bounded tables and
  configured SQL connection.
- **Standard:** reuses the existing database pool contract; creates at most
  the existing 100 collection schemas, each bounded to 64 fields.
- **Distributed:** sync remains operator-driven against the target database;
  there is no cache, worker, or inter-node coordination.

## Database impact

- **SQLite:** uses the existing `hydro_collections` schema and Store adapter;
  no migration or SQL generated from Blueprint names is added.
- **PostgreSQL:** uses the same certified Store adapter and tenant transaction
  scope; no new RLS policy or physical schema is introduced.
- **Future adapters:** this CLI accepts only currently supported SQLite and
  PostgreSQL dialects; adapter certification remains required before extension.

## Cache impact

No cache keys, TTLs, invalidation, stampede behavior, fallback, or cache service
changes.

## Deployment and compatibility

The command is additive. Older binaries ignore collection schemas they do not
use; rollback does not delete schema metadata or records. The Collections
module's existing migration remains the only storage prerequisite. Schema
changes after import are reviewed and applied in Admin Studio; sync refuses to
overwrite them. No new migration metadata is required because no migration is
created.

## Validation

- Unit tests cover supported mappings, rejected features, permissions guard,
  relationship order/cycles, drift, idempotency, and partial-write retry
  behavior.
- SQLite integration exercises sync through the existing SQL Store and
  migration.
- PostgreSQL contract ran against an isolated schema in a PostgreSQL 18 database
  using a non-superuser `NOBYPASSRLS` role.
- `TMPDIR=/var/tmp/hydro-go-tests go test ./... -count=1 -p 1 -timeout=5m`
  passed with the isolated PostgreSQL test DSN; `go vet ./...`, moduleguard,
  migrationguard, Edition importguards, and `git diff --check` passed.
- `go test ./...`, `go vet ./...`, moduleguard, migrationguard,
  `hydro project check`, and import guards remain required.

## References

- [Dynamic collections validation](../validation/dynamic-collections-admin-2026-10-07.md)
- [Admin Studio product audit and plan](../planning/ADMIN_STUDIO_PRODUCT_AUDIT_AND_PLAN.md)
- `cmd/hydro/collections_cmd.go`
- `internal/collections/sqlstore/store.go`
