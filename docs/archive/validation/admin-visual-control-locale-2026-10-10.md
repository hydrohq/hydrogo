# Admin visual-control locale verification — 2026-10-10

The theme toggle updates its accessible name after initialization and family/mode changes. It now uses the Admin document language. Theme-family and color-mode confirmation toast copy also follow that language.

## Evidence

- `GOCACHE=/tmp/hydrogo-adminstudio-gocache go test ./internal/ui -run TestAdminVisualControlsFollowAdminLanguage -count=1` — passed.
- Full `internal/ui` test suite and vet — passed.
- Admin-management Playwright test discovery — passed; the browser case now covers English/Turkish toggle naming and appearance/theme feedback.
- Live browser execution remains blocked at the fixture's loopback listener (`listen EPERM`), before application assertions.
- `git diff --check` — passed.
