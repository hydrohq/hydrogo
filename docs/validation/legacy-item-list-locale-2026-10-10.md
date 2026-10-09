# Legacy Admin item-list locale verification — 2026-10-10

The shared item-list initializer now selects fallback placeholder and remove-button accessible-name copy from the Admin document language. The shared media-preview initializer also localizes its empty state and image alt text when those use the legacy Turkish defaults; explicit component copy remains intact. The legacy renderer passes its field label so dynamically created rows have useful accessible names.

## Evidence

- `GOCACHE=/tmp/hydrogo-adminstudio-gocache go test ./internal/ui ./internal/adminstudio ./modules/news/transport ./modules/pages/transport -count=1` — passed.
- `./node_modules/.bin/playwright test custom/admin-management.spec.ts --list` from `e2e/` — passed; the new locale case is discovered.
- Focused Playwright case includes English/Turkish item-row names and dynamic media empty-state/alt text. Runtime execution — blocked during fixture setup: `spawnSync go EPERM` while invoking `go run ./cmd/hydro migrate`. The browser did not reach the application and no browser assertions passed.

This is local SQLite/package-test evidence only. Browser runtime and PostgreSQL acceptance remain open gates.
