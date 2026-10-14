# Admin media library mobile layout — 2026-10-10

## Change

The Admin media grid now uses one column at widths up to 520px. Media card buttons and selects receive 44px minimum touch height at tablet/mobile breakpoints, including the copy and visibility controls. The authenticated media upload browser journey now checks the 390px layout, touch target heights, and WCAG 2.1 A/AA before restoring desktop width.

## Verification

- `go test ./internal/ui -run '^TestMobileMediaLibraryUsesSingleColumnAndTouchSizedControls$' -count=1 -p 1` — passed.
- `go test ./internal/ui -count=1 -p 1` — passed.
- `go vet ./internal/ui` — passed.
- Full root SQLite suite `go test ./... -count=1 -p 1 -timeout=8m` — passed across all packages.
- Full root `go vet ./...` — passed.
- `playwright test custom/admin-management.spec.ts --list` — discovered all 5 cases.
- Targeted Playwright runtime was attempted with a writable Go cache. It stopped before browser assertions because this environment rejects binding the E2E server to `127.0.0.1` (`listen EPERM`).
- `git diff --check` — passed.

The mobile layout and accessibility browser assertions are added but require rerunning in an environment that permits the fixture server to bind loopback.
