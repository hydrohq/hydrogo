# Admin session warning locale and startup validation — 2026-10-10

The session warning script is emitted in the document head. Its previous
startup checked for `document.body` immediately and returned before the body
existed, so the warning never initialized. Startup now waits for
`DOMContentLoaded`, then loads session expiry data. Warning title, body, extend
action, and retry action follow the active Admin document language (English or
Turkish).

Validation passed:

```sh
GOCACHE=/tmp/hydrogo-adminstudio-gocache go test ./internal/ui \
  -run 'TestAdmin(SessionWarningUsesAdminLocale|ShellIncludesSessionExpiryWarning)' -count=1
GOCACHE=/tmp/hydrogo-adminstudio-gocache go vet ./internal/ui
cd e2e && ./node_modules/.bin/playwright test custom/admin-management.spec.ts --list
git diff --check
```

The Playwright locale/accessibility scenario was attempted. It did not reach
application startup because this environment denies binding `127.0.0.1`
(`listen EPERM`). Therefore browser rendering, the expiry dialog interaction,
and axe assertions remain unverified here.
