# Generated Admin destructive action confirmation — 2026-10-08

Generated record deletion and draft discard now use the Admin Studio's existing
typed `data-confirm-action` confirmation contract. This prevents accidental
destructive actions while leaving authorization, CSRF, and persistence behavior
on their existing server-side paths.

## Verification

- `TMPDIR=/var/tmp/hydro-go-tests go test ./... -count=1 -p 1 -timeout=8m` passed.
- The generated Project Edition Chromium CRUD journey passed. It dismissed the
  project deletion prompt and verified the record remained, then accepted it
  and verified the record was removed.
- The generated editorial Chromium journey passed (7/7), including dismissing
  draft discard and verifying the draft remained before accepting discard.
- `git diff --check` passed.

This change adds no routes, migrations, dependencies, or new browser runtime.
