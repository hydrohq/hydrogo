# ADR-0332: Reversible content deletion before permanent erasure

- **Status:** Accepted
- **Date:** 2026-09-29
- **Related:** [ADR-0013](0013-identity-and-security-boundaries.md), [ADR-0014](0014-publication-routes-and-erasure-recovery.md), [ADR-0011](0011-consistency-upgrade-and-recovery.md)

## Decision

Pages, news, form inbox entries, and media assets use nullable `deleted_at`
timestamps for a recoverable 30-day trash window. Normal reads and public
routes exclude trashed rows. Restore clears the timestamp only within
retention. Permanent deletion is an erasure operation and must first record an
intent through the established erasure boundary. Restoring trash never
reverses an erasure intent.

Trash access is tenant-scoped and permission-checked. Media bytes and
derivatives remain present while the asset is recoverable; unused-media GC
skips trashed rows. Hard deletion is a terminal retention/erasure operation,
not an ordinary admin delete.

## Compatibility and operations

Four additive migrations belong to their owning optional modules, so an
inactive module creates no schema. SQLite and PostgreSQL use nullable
timestamps; MySQL uses nullable `DATETIME(6)`. Old binaries ignore the new
columns. Rollback is binary-only until expired rows are permanently erased;
dropping the columns is not a data-preserving rollback.

Retention sweeps must be capped and idempotent. They record/check the erasure
intent before removing form PII or media bytes. If the erasure log is
unavailable, permanent removal is deferred. Capacity impact is additive
timestamp/index storage plus up to 30 days of retained media bytes, which
operators must include in the media budget. Cleanup uses the existing bounded
maintenance scheduler and does not start a worker for an inactive module.

## Verification gates

SQLite and PostgreSQL tests cover visibility, tenant isolation, restore,
expiry, and idempotent purge. Media tests verify trash assets are not collected
by unused-media GC. Admin actions must use typed Templ components and existing
authorization/CSRF contracts. Production permanent deletion stays disabled
until it is wired to the verified external erasure log required by ADR-0014.
