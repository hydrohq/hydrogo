# ADR-0308: Admin Login Security Notice

## Status

Accepted.

## Context

Staff need a concise way to identify the previous successful account login and learn whether failed password attempts occurred since then. Unknown email addresses must remain indistinguishable, and the dashboard must not expose another account's activity.

## Decision

Persist current last-login time and IP, a failed password-attempt counter, and a snapshot of the prior session's login and failed-attempt count. Only password checks against an existing account increment its counter; unknown addresses do not create records or alter responses. After a successful password and MFA check, the system atomically snapshots the previous login and attempts, records the new login, and resets the current counter. The login response shows a brief summary and the Admin Studio dashboard reads only the signed-in account's tenant-scoped snapshot.

The dashboard notice reports time, IP address from the prior login, and the number of failed attempts. It advises the account owner to change their password when attempts occurred. It does not include email addresses, request bodies, or internal authentication errors.

## Database and rollout

Migration `202609290007_auth_login_security` adds nullable login timestamps, bounded IP strings, and non-negative integer counters. SQLite, PostgreSQL, and MySQL retain their native time and integer types. Existing accounts start with empty history. The expansion migration is readable and writable at schema epoch 1; older binaries ignore the new columns. Rollback is forward-only and retains the additive columns.

## Capacity and security

The tracker uses the existing `users` row, so it adds no worker, cache, route, or service. Tenant identity predicates and PostgreSQL RLS remain enforced by the existing user store transaction boundary. Failed-attempt writes use an atomic counter update, and successful session rotation reads and writes the summary in one transaction. No raw password data or unknown-account identifiers are stored.
