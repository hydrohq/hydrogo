# ADR-0327: Bounded Admin Audit Explorer

- **Status:** Accepted
- **Date:** 2026-09-29
- **Supersedes:** None
- **Superseded by:** None

## Context

HydroGo already writes append-only audit events, but operators need a searchable
Admin Studio view. Read access must preserve tenant boundaries and the PII
rules in [ADR-0075](0075-audit-access-and-pii-boundary.md).

## Decision

The audit explorer is an admin-only, read-only page. Queries require the
current tenant when tenant scoping is enabled, are parameterized, and return at
most 100 rows. Filters cover actor, action, resource, target, and an inclusive
date range. Detail JSON is bounded to 4 KiB per side and escaped by Templ.
Audit records remain append-only. The explorer does not capture page bodies,
form contents, credentials, or other arbitrary PII. Existing records remain
readable and are marked as legacy when they predate the added metadata.

The initial migration adds actor/resource/time and before/after JSON metadata
with backward-compatible defaults. It is additive and forward-only; recovery
preserves the audit log and uses a corrective migration rather than deleting
history. New audit writers may attach only explicitly selected, bounded
metadata. Page and news create/update/publish writers record only the old and
new entity version; sectoral writes the resulting entity version. Other event
writers retain empty state objects until they provide safe domain-specific
details. No content fields are captured in this initial implementation.

## Alternatives

- Adding a separate audit service: rejected; the existing immutable log is the
  source of truth.
- Capturing full record snapshots: rejected because this can retain sensitive
  data and create unbounded storage growth.
- Loading an unbounded history page: rejected due to memory and query cost.

## Consequences

### Benefits

- Admins can search and page through existing events without mutating them.
- Tenant filtering is enforced in the data query, not just the UI.

### Costs and risks

- Older rows have only compatibility defaults and cannot show historical diffs.
- Detailed field diffs require each domain writer to select safe fields
  explicitly; the current diff view shows version transitions for page/news.

## Scale impact

- Compact: indexed bounded reads only; no worker or cache is added.
- Standard: tenant/time index supports recent history paging.
- Distributed: the same SQL contract applies; no external service is required.

## Database impact

- SQLite: additive columns and tenant/time index; migration is transactional.
- PostgreSQL: same logical contract; migration is transactional.
- MySQL: bounded VARCHAR metadata columns and index; migration is non-transactional.

## Cache impact

No cache is used. Audit reads must reflect the latest committed events.

## Deployment and compatibility

The migration is expand-only and defaults old records to empty metadata and an
epoch timestamp. Old binaries ignore the additional columns; new binaries
require the migration before serving the explorer. Rollback preserves history
and is performed through a forward corrective migration.

## Validation

SQLite tests cover writes, filtering, paging, tenant isolation, missing scope,
and bounded JSON. Migration metadata is checked by the migration guard. The
full Go suite validates the shared query and UI integration.

## References

- [ADR-0075](0075-audit-access-and-pii-boundary.md)
- [Audit explorer migration](../../migrations/202609291001_audit_explorer_fields/migration.yaml)
