# Generated Admin save error recovery — 2026-10-09

## Change

Generated collection create/update API transport or 5xx failures return HTTP
502 with the original edit form rendered in the shared Admin shell. The error
alert is localized and accessible. User-entered values, CSRF token, and save
button are preserved, so a transient problem can be retried without retyping.
If a delete request fails and the record is still readable, the Admin restores
its edit form with the current record, CSRF token, delete action, and a
localized alert that tells the user to check the list before retrying. If the
record cannot be fetched (including a delete that completed despite an
ambiguous response), the handler returns delete-specific localized guidance
without exposing the API response.

## Verification

- A fake API returns an HTTP 500 containing a PostgreSQL URL, internal host,
  filesystem path, and panic marker.
- Create, update, and delete cases, in both Turkish and English, show the expected
  localized `role="alert"` inside the shared shell.
- Submitted create/update values remain in their forms. Delete restores the
  current record and delete action with CSRF. No internal detail is returned,
  and no success redirect occurs.
- Focused test passed:
  `go test ./internal/adminstudio ./internal/i18n -run '^TestGeneratedAdminLocalizesAndSanitizesResourceFailure$' -count=1 -p 1 -timeout=4m`.
- Delete fallback test passed:
  `go test ./internal/adminstudio ./internal/i18n -run '^TestGeneratedAdminDeleteFailureWithoutReadableRecordUsesDeleteCopy$' -count=1 -p 1 -timeout=4m`.
- Full package tests passed:
  `go test ./internal/adminstudio ./internal/i18n -count=1 -p 1 -timeout=5m`.
- `go vet ./internal/adminstudio ./internal/i18n` and `git diff --check` passed.

## Browser retry coverage added — 2026-10-09

The generated rich-text E2E fixture can now fail the next article create once.
The Playwright scenario asserts the 502 response, preserved title and rich-text
body, localized accessible alert, absence of fixture internals, axe results,
successful retry, and API readback. The Go fixture package compiled and
Playwright discovered the case (8 tests total). An actual browser execution was
attempted, but the runner failed before the scenario started because this
environment denies loopback listeners (`listen EPERM: operation not
permitted 127.0.0.1`). Therefore browser behavior is still pending a permitted
runner; discovery is not counted as execution.

## Limits

This is an in-process HTTP test. Browser interaction, PostgreSQL, and deployed
Edition error mapping remain open. The delete fallback is covered for Turkish
and English when the record is no longer readable.
