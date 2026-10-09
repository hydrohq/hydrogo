# Admin MFA login accessibility validation — 2026-10-09

The real Admin login MFA challenge now hides the password-only email, password,
remember-me and forgot-password controls after the password is accepted. The
challenge and invalid-code alert are checked by axe-core against WCAG 2.1 A/AA
in light and dark themes. This caught the dark warning text using a light-theme
color; the warning now uses theme-specific contrast tokens. The browser test
also asserts that the password controls are hidden during the challenge.

The MFA challenge and invalid-code states passed in both theme modes on the
SQLite fixture and PostgreSQL 18.6 fixture (1/1 Playwright test per dialect).
Each run covers the actual password → MFA challenge → invalid code → valid TOTP
journey and then the existing MFA account, role, permission, backup, and Admin
route checks. The viewer-login fixture now honors a `429 Retry-After` response
before retrying so the added invalid-code attempt does not make the whole test
flaky under the shared per-IP login limiter.

`go test ./internal/ui ./internal/auth -count=1`, `git diff --check`, and the
Impeccable detector also passed; the detector reported no findings. The MFA
challenge and incorrect-code alert are covered, but MFA-code throttling,
account lockout, and deployed identity-provider flows remain open.

## Login IP rate-limit feedback

The browser test exhausts the real login IP limiter through a separate HTTP
client, confirms the server returns `429`, `Retry-After: 60`, and
`Cache-Control: no-store`, then submits the visible Admin login form and checks
the localized alert. After waiting for the advertised window, it repeats the
flow in Turkish/dark mode. Both rate-limit alerts passed axe WCAG 2.1 A/AA.
The complete MFA/Admin browser scenario passed 1/1 on SQLite and 1/1 on
PostgreSQL 18.6. This covers login IP limiter presentation only; MFA-code
throttling, account lockout, and deployed identity flows remain unverified.
