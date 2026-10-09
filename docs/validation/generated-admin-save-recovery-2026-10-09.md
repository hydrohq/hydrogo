# Generated Admin save error recovery — 2026-10-09

## Change

Generated collection create/update API transport or 5xx failures return HTTP
502 with the original edit form rendered in the shared Admin shell. The error
alert is localized and accessible. User-entered values, CSRF token, and save
button are preserved, so a transient problem can be retried without retyping.
Delete operations keep the sanitized short error response because there is no
edit form to restore.

## Verification

- A fake API returns an HTTP 500 containing a PostgreSQL URL, internal host,
  filesystem path, and panic marker.
- Create and update cases, in both Turkish and English, show the expected
  localized `role="alert"` inside the shared shell.
- The submitted title remains in the form, the save form and CSRF token are
  present, no internal detail is returned, and no success redirect occurs.
- Focused test passed:
  `go test ./internal/adminstudio ./internal/i18n -run '^TestGeneratedAdminLocalizesAndSanitizesResourceFailure$' -count=1 -p 1 -timeout=4m`.
- Full package tests passed:
  `go test ./internal/adminstudio ./internal/i18n -count=1 -p 1 -timeout=5m`.
- `go vet ./internal/adminstudio ./internal/i18n` and `git diff --check` passed.

## Limits

This is an in-process HTTP test. Browser interaction, PostgreSQL, delete
failure presentation, and deployed Edition error mapping remain open.
