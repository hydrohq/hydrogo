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

## Limits

The handler still returns a plain error response instead of re-rendering the
form in the shared Admin shell. Browser interaction, PostgreSQL, and deployed
Edition error mapping remain open gates.
