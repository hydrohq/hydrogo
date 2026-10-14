# Admin extension channel route security — 2026-10-09

## Change

The bind and revoke route handlers now live in testable functions used by the
production route table. They require an active Admin whose server-derived
session belongs to that user. The tenant is resolved from the trusted request
context; actor and tenant data in JSON are ignored. The routes remain
`router.Protected` and continue through the global CSRF middleware.

## Verification

- Missing CSRF and a valid-CSRF Editor request return 403 without calling the
  channel registry.
- An Admin with a session for another user is rejected before registry access.
- A valid Admin can bind and revoke; the registry receives the server-derived
  user, session and tenant principal.
- Forged `user_id`, `session_id`, and `tenant_id` JSON properties do not alter
  that principal.
- Registry bind errors return 403 without leaking internal detail.
- Focused test passed:
  `GOCACHE=/tmp/hydrogo-go-cache go test ./cmd/hydro -run '^TestExtensionChannelMutationRoutesEnforceRoleCSRFAndSessionBinding$' -count=1 -p 1 -timeout=4m`.
- Full `cmd/hydro` tests, `go vet ./cmd/hydro`, and `git diff --check` passed.

## Limits

This verifies the real route handler and middleware composition with an
in-process registry fake. Existing ChannelManager contract tests separately
cover challenge binding, permission enforcement, replay, expiry and revocation.
The extension channel flow has not been exercised in an authenticated browser
or a deployed Edition.
