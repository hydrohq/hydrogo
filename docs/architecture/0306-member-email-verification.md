# ADR-0306: Member Email Verification Links

- Status: Accepted
- Date: 2026-09-29

## Context

Member accounts use a separate identity table from Admin Studio staff. Staff
recovery tokens reference `users` and must not be reused for members.

## Decision

When the Members and Mail capabilities are active, an administrator can
request a fresh verification email for an unverified member. A random token is
stored only as a hash in a tenant-scoped table and expires after 24 hours.
Issuing a new token consumes older outstanding tokens for that member.
Verification consumes the token and marks the member verified in the same
transaction. The resend endpoint is rate limited and audited.

## Compatibility and operations

The additive migration is owned by the Members module and adds no route,
worker, or table when that module is inactive. SQLite stores UTC RFC3339 text;
PostgreSQL stores `TIMESTAMPTZ`; MySQL stores `DATETIME(6)`. All dialects use
the same tenant discriminator, unique token hash, member foreign key, and
lookup index. Expired/consumed token rows are small and bounded per member by
the resend rate limit and revocation of older outstanding tokens.

The migration depends on member metadata and tenant-scope migrations. There
is no production down migration; recovery uses the supported backup/forward
migration path. SQLite/PostgreSQL migrations are transactional, while MySQL
uses the repository's nontransactional DDL metadata.
