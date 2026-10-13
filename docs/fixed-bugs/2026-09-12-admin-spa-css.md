# Admin Studio SPA CSS Initial Navigation Fix

## Symptoms

Within Admin Studio, when navigating to `API & Integrations`, `Operations`, `Security`, `Permissions`, or `Resource Usage` pages via the sidebar for the first time, page-specific CSS failed to apply, causing content to appear as an unstyled block. Reloading the browser resolved the styling.

## Root Cause

Admin pages were loaded by the SPA router as fragments from new HTTP responses. Styles inside the fragment carried the CSP nonce of the new response, whereas the active document's CSP policy only accepted the initial page's nonce. Additionally, re-serializing the fragment via `innerHTML` hid the browser's nonce attribute and dropped the normalized nonce.

## Applied Fixes

- The SPA router normalizes fragment styles with the active document's CSP nonce prior to inserting them into the DOM.
- Fragment DOM transfers use `replaceChildren(...Array.from(newMain.childNodes))` rather than `innerHTML` serialization, preserving nonce values across DOM operations.
- Missing styles for resource consumption, developer panels, and the Permissions matrix were unified into the global Admin Studio CSS bundle.
- Permissions table, badges, rules cards, and mobile overflow behaviors were integrated into global CSS.
- E2E test manifests were expanded to verify actual sidebar SPA navigation transitions rather than direct URL loads only.
- Computed-style assertions verify that grid and table styles are applied across these routes.

## Evidence

- `internal/ui/admin_shell.go`
- `internal/ui/admin_areas.go`
- `tools/e2egen/main.go`
- `e2e/manifest.json`
- `TASKS.md`

Verification outcomes:
- `go test ./internal/ui ./tools/e2egen -p 1 -count=1` passed.
- `go vet ./...` passed.
- Chromium Playwright E2E passed.

## Operator Note

To apply changes to a running development server:

```powershell
hydro run
```

Then clear browser cache with `Ctrl + F5`.
