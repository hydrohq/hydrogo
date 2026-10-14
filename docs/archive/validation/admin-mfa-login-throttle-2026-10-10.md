# Admin MFA login throttling validation — 2026-10-10

Added HTTP-level regression coverage for the existing MFA attempt limiter on
the Admin password-plus-code login path. With a two-attempt window, two invalid
codes return the normal unauthorized response and produce two audit events; the
next request returns `429` with `Retry-After`, does not invoke MFA code
verification, and never creates a session.

Validation passed:

```sh
GOCACHE=/tmp/hydrogo-adminstudio-gocache go test ./internal/auth \
  -run 'TestHTTPLogin(MFAAttemptsAreThrottledBeforeVerificationAndSessionCreation|RequestsAuthenticatorCodeWithServerClock)' -count=1
GOCACHE=/tmp/hydrogo-adminstudio-gocache go vet ./internal/auth
```

This proves the in-process HTTP throttle contract only. It does not prove
cross-instance/distributed rate limiting, persistent account lockout, browser
interaction, or deployed identity behavior. No persistent lockout behavior is
introduced or implied.
