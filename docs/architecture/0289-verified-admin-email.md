# ADR-0289: Verified administrative email ownership

Status: Accepted

## Context

The test-mail preview must send only to a mailbox the signed-in account has proven it controls. A valid address syntax and an authenticated admin session do not prove mailbox ownership.

## Decision

- The optional auth schema stores a nullable `users.email_verified_at` timestamp.
- Accepting an emailed staff invitation marks the address verified. OAuth may mark it verified only when the provider supplies a verified claim matching the existing account address.
- Existing accounts request a one-use, two-hour confirmation link. It reuses the auth recovery-token table with a separate purpose and applies the user/tenant transaction boundary. Email changes clear the verification timestamp and revoke outstanding confirmation tokens.
- The test-email endpoint requires the active admin's stored verification timestamp. The preview page exposes the confirmation action until that requirement is met.
- Inactive auth does not add a route or migration. No worker or new token table is introduced.

## Operations and compatibility

SQLite, PostgreSQL and MySQL receive a nullable timestamp column through an additive migration. Old accounts remain unverified until confirmed. Existing readers can continue during rollout because the field is nullable; the new writer requires the migration before the new binary is enabled. Rollback is forward-only because verification evidence should not be discarded.

## Security and lifecycle

Confirmation links use random one-time secrets stored as digests, expire after two hours, are rate limited, sent only through the configured mail sender, and are not returned in HTTP responses. Changing an account email invalidates prior confirmation links and confirmation state. Confirmation endpoints use no-store and no-referrer headers.
