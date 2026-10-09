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

`go test ./internal/ui -count=1`, `git diff --check`, and the Impeccable detector
also passed; the detector reported no findings. This evidence covers the MFA
challenge and incorrect-code alert, not all lockout/rate-limit responses or
deployed identity-provider flows.
