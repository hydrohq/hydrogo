# Generated array dynamic controls and interface locale — 2026-10-10

Generated Array fields now carry localized row placeholder and accessible-name
patterns alongside the existing add-item copy. The Admin shell applies those
patterns whenever rows are added or removed, and renumbers both input and
remove-button accessible names after a row is deleted. Direct uses of the typed
component without locale props retain Turkish defaults. This changes only
presentation metadata; it adds no route, HTMX behavior, data persistence, or
migration.

English and Turkish catalog entries are passed through the generated Admin
handler into the typed `ItemListEditor`. The generated-richtext Playwright case
adds/removes an English row, checks its placeholder and row-specific names,
then repeats the checks in Turkish. Test discovery passed (1/1). Its execution
was attempted but stopped before the server/browser assertions because the
fixture could not bind `127.0.0.1` (`listen EPERM`). This case is present but is
not a passing E2E result in this environment.

Verification passed:

```sh
GOCACHE=/tmp/hydro-admin-gocache go test -count=1 -p 1 -timeout 90s ./internal/ui ./internal/ui/components ./internal/adminstudio ./internal/i18n
GOCACHE=/tmp/hydro-admin-gocache go test ./... -count=1 -p 1 -timeout=5m
GOCACHE=/tmp/hydro-admin-gocache go vet ./...
cd e2e && npm exec playwright test custom/generated-richtext.spec.ts -- --project=chromium --workers=1 --grep 'widget controls follow' --list
```

The focused and full root Go suites passed; Playwright discovery listed one
scenario. Browser execution above reached the case but listener setup failed
before assertions. The Impeccable detector reported six existing Admin shell
CSS warnings (two thick side borders, an Inter font warning, and three width
transitions). The changes in this stage do not touch those CSS rules; the
visual design cleanup remains open for a separate inspected pass.
