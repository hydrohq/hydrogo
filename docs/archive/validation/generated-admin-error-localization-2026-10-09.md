# Generated Admin operational error localization — 2026-10-09

## Finding and change

The generic generated CRUD API error was always English, even when the
interface locale was Turkish. It now uses
`admin.generated.error.operation` from the active Admin locale catalog. All
generated API error call sites pass their request to the shared formatter.

## Verification

- A fake API returns HTTP 500 with a PostgreSQL URL, internal host,
  filesystem path, and panic marker.
- Turkish and English Admin locale cookies receive their respective safe
  message, HTTP 502, and no success redirect.
- The response does not contain any injected internal API detail.
- The focused test passed:
  `go test ./internal/adminstudio ./internal/i18n -run '^TestGeneratedAdminLocalizesAndSanitizesResourceFailure$' -count=1 -p 1 -timeout=4m`.
- Full package tests passed:
  `go test ./internal/adminstudio ./internal/i18n -count=1 -p 1 -timeout=5m`.
- `go vet ./internal/adminstudio ./internal/i18n` and `git diff --check` passed.

## Follow-up

Create/update failures now re-render the form with a localized accessible
alert and preserve the submitted values for retry. See
[`generated-admin-save-recovery-2026-10-09.md`](generated-admin-save-recovery-2026-10-09.md).

Browser interaction, PostgreSQL, delete-failure interaction, and deployed
Edition error mapping remain open gates.
