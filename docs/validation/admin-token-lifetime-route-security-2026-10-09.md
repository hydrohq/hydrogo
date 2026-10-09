# Admin token lifetime route security — 2026-10-09

The token lifetime settings mutation is now tested through the route
classification and CSRF middleware composition used by the application. The
test registers `POST /admin/security/token-lifetimes` as `router.Protected` and
wraps it with `security.CSRFMiddleware`.

## Evidence

- A mismatched form token returns 403 before the settings store is written.
- A valid token from an Editor still returns 403 and does not write settings.
- A valid token from an Admin saves the bounded durations and redirects.
- Unsupported durations return 400 without an additional store write.
- Focused test passed:
  `GOCACHE=/tmp/hydrogo-go-cache go test ./cmd/hydro -run '^TestAdminTokenLifetimeSettingsApplyAndRejectUnsupportedChoices$' -count=1 -p 1 -timeout=4m`.
- Full `GOCACHE=/tmp/hydrogo-go-cache go test ./cmd/hydro -count=1 -p 1 -timeout=8m` passed.
- `GOCACHE=/tmp/hydrogo-go-cache go vet ./cmd/hydro` and `git diff --check` passed.

## Limits

This test proves handler behavior behind the registered route class and CSRF
middleware, but it does not start the production server or test an authenticated
browser session. Other platform Admin mutation routes remain in the broader
role/permission/CSRF matrix.
