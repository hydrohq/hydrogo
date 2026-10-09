# Admin responsive matrix authentication coverage — 2026-10-10

The shared responsive matrix previously declared protected Admin surfaces but
never iterated that inventory. Its one `/admin` check also did not assert an
authenticated response, so a redirect to `/login` could satisfy the width
assertion. The keyboard test also treated any ordinary element border as a
visible focus indicator. The matrix now iterates Admin login, dashboard, page
editor, news editor, and media library at 390, 768, and 1280 CSS pixels. The
focus assertion now requires an actual nonzero visible outline. Protected cases
skip explicitly without `HYDRO_E2E_STORAGE_STATE`; with a configured state they
require HTTP 200, a non-login URL, and a rendered `data-studio-role` before
checking horizontal overflow.

Validation:

- `cd e2e && ./node_modules/.bin/playwright test custom/a11y-responsive.spec.ts --list` — 28 cases discovered, including all 15 Admin viewport/surface pairs and the stricter focus assertion.
- Runtime browser assertions remain unverified in this environment: the
  fixture requires a reachable local server, and prior attempts to bind its
  loopback listener returned `listen EPERM` before the app started.
- Impeccable detector ran against `internal/ui/admin_shell.go`. It reported
  six warnings: two side-tab style matches, one common Inter font match, and
  three width-transition matches. These are heuristic findings; no visual
  severity is assigned without inspecting the rendered surfaces. The product
  UI follows the separate Admin visual-family/token contracts in ADR-0341 and
  ADR-0053.

This closes a false-positive path in test coverage; it does not certify visual
quality, axe results, actual viewport rendering, or keyboard behavior. Those
remain open until the browser suite can run against an authenticated Edition.
