# R3 identity and MFA foundation validation

**Date:** 2026-09-08  
**Roadmap:** R3 security foundation  
**Status:** Foundation gate succeeded; commercial pilot security gate remains open.

## Validated behaviors

- When a user is disabled, all their DB-backed session records are revoked.
- A disabled user's session cannot be re-validated.
- The TOTP secret is stored encrypted via AES-GCM using the master key.
- Recovery codes are stored only as SHA-256 hashes and are single-use via atomic update.
- SQLite and PostgreSQL auth contract tests exercise MFA enroll/enable/verify and recovery code reuse.
- An admin-controlled user disable HTTP route and audit event were added.
- Password reset tokens carry purpose, hash, expiry, and single-use constraints; the reset applies password change and session revoke in the same transaction.
- Login and MFA attempts are rate-limited by separate, bounded process-local limiters; MFA login/enable attempts consume the MFA budget, and when the limiter capacity is full, new keys evict old records.
- The password reset request/confirm HTTP contract was added; the request endpoint returns `202 Accepted` even for unknown accounts, the token is provided only to the injected notifier, and confirm executes session revocation within the transaction using the single-use token.
- Invite tokens are stored only as SHA-256 hashes, can only be generated for the editor/viewer role, and upon acceptance, token consumption and user creation are bound to the same transaction; the same invite cannot be accepted a second time.
- After an MFA-enabled user authenticates with a password, they cannot obtain a session without an MFA code; TOTP or single-use recovery code is validated in the login contract test.
- Encoded parameters of Argon2 password hashes are validated within bounded memory/iteration/parallelism ranges; an oversized or malformed hash is rejected during login without turning into resource consumption.
- If the MFA store dependency is missing, the service does not panic; the `MFAStore` requirement is applied fail-closed as a separate error, and the login flow does not mistakenly consider MFA disabled.

## Executed checks

```text
go test ./...
go vet ./...
go build ./cmd/hydro
```

The MFA master key value is fetched from a secret source via `HYDRO_MFA_MASTER_KEY`; the plaintext value is not written into the config or manifest.

## Open scope

The multi-instance contract of the limiter, production notifier adapter, security notification, and restore/blue-green scenarios on a clean environment are subsequent gates for R3.
