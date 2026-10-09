# Admin command palette locale — 2026-10-10

The shared Admin command palette now renders its built-in command copy, search labels, keyboard help, and no-results state in the active Admin language. Query case normalization follows that same language. Role and capability filtering still runs before command rendering.

## Evidence

- `GOCACHE=/tmp/hydrogo-adminstudio-gocache go test ./internal/ui -run TestSpotlightSearchUsesAccessibleDialogAndGlobalShortcuts -count=1` — passed.
- Admin-management Playwright test discovery — passed; the role-session case covers English/Turkish palette labels, command copy, and locale-aware uppercase queries.
- `git diff --check` — passed.
- Live browser assertions remain unverified because the fixture server cannot bind `127.0.0.1` in this execution environment (`listen EPERM`).
