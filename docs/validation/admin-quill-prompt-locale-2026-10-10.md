# Quill image URL prompt locale — 2026-10-10

When an Edition does not provide `HydroOpenMediaPicker`, the Quill image toolbar falls back to a native URL prompt. That prompt now follows the Admin document language.

## Evidence

- `GOCACHE=/tmp/hydrogo-adminstudio-gocache go test ./internal/ui -run TestAdminQuillImageURLFallbackUsesAdminLanguage -count=1` — passed.
- `GOCACHE=/tmp/hydrogo-adminstudio-gocache go vet ./internal/ui` — passed.
- `git diff --check` — passed.

This verifies the rendered shell contract, not a live browser prompt interaction. The Admin browser fixture cannot bind `127.0.0.1` in this environment (`listen EPERM`); see the wider runtime gate in the Admin Studio plan.
