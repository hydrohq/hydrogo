# R5 SDK artifact grant contract — 2026-09-16

**Status:** Successful (local/WSL)

`internal/sdkdist.ArtifactGrant` and `ValidateGrant` define the short-lived authorization payload between the broker and CLI. The grant binds project ID, Core/SDK versions, artifact URL, issued/expiry timestamps, and access token. The token is never persisted or logged; expiry is at most 24 hours, clock skew tolerance is 5 minutes, and the URL additionally passes through download policy.

## Evidence

- Windows: `go test ./internal/sdkdist` passed.
- WSL Ubuntu/Go 1.27: `go test ./internal/sdkdist` and `go test ./...` passed.
- Incorrect project/version, expired, post-dated, long-lived, and unsafe URL grants were rejected.
- `RedactToken` prevents short or full token leakage.

## Scope & Boundaries

This is not a broker or OAuth implementation. Token issuance, user session, revocation, and entitlement checks remain in external services; the CLI only validates broker-issued grants against this contract.
