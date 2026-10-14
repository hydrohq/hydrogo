# ADR-0183: Bind forms and media pool operations to tenant transactions

- Status: Accepted (forms/media stores covered; remaining direct pool paths open)
- Date: 2026-09-20
- Related decisions: [ADR-0172](0172-tenant-scoped-site-data.md), [ADR-0165](0165-tenant-scoped-media-metadata.md), [ADR-0180](0180-postgres-tenant-rls-enforcement.md)

## Decision

When the forms registry or media metadata store receives a raw `*sql.DB`, it
opens a short tenant-aware transaction before executing the operation. Existing
transaction executors are reused unchanged, so callers that already participate
in a larger atomic workflow do not acquire a nested transaction. Row-producing
operations use `WithinQuery` and scan before the transaction commits.

## Consequences

- Form schema reads/list/upsert/delete and media asset/variant reads and
  processing updates share the PostgreSQL RLS binding with their queries.
- Existing explicit tenant predicates remain mandatory application-level
  defense.
- Raw pool paths in other transport/projection and commerce code still require
  inventory and conversion before shared SaaS certification.
