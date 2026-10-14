# Admin Studio and Customer Flows Chromium Matrix

**Date:** 2026-10-07  
**Result:** 64/64 Chromium tests passed  
**Scope:** Admin and editor experience, generated/dynamic CRUD, MFA, accessibility,
responsive behavior, and public pages.

## Execution Environment

The Playwright test suite was executed against a seeded, isolated SQLite Edition with a valid, single-use
admin storage state. Generated Admin smoke tests use the
`HYDRO_E2E_BASE_URL` and `HYDRO_E2E_STORAGE_STATE` environment variables;
an arbitrary unauthenticated `localhost:8090` server is not a valid fixture for the full Admin
matrix. `HYDRO_E2E_ADMIN_EMAIL` and
`HYDRO_E2E_ADMIN_PASSWORD` are intended solely for the isolated local test account.

Isolated server preparation:

```sh
go build -trimpath -buildvcs=false -o "$TMPDIR/hydro" ./cmd/hydro
export HYDRO_MEDIA_DIR="$TMPDIR/media"
export HYDRO_ENABLE_DEMO_FRONTEND=true
"$TMPDIR/hydro" migrate --database-dialect sqlite --database-dsn "$TMPDIR/edition.sqlite"
"$TMPDIR/hydro" seed-demo --confirm-demo --database-dialect sqlite --database-dsn "$TMPDIR/edition.sqlite"
printf '%s\n' "$HYDRO_E2E_ADMIN_PASSWORD" | "$TMPDIR/hydro" create-admin --email "$HYDRO_E2E_ADMIN_EMAIL" --password-stdin --database-dialect sqlite --database-dsn "$TMPDIR/edition.sqlite"
"$TMPDIR/hydro" serve --dev --database-dialect sqlite --database-dsn "$TMPDIR/edition.sqlite" --address 127.0.0.1:PORT
```

While the server is running, after generating the temporary `auth-state.json`:

```sh
cd e2e
export HYDRO_E2E_BASE_URL=http://127.0.0.1:PORT
export HYDRO_E2E_STORAGE_STATE="$TMPDIR/auth-state.json"
npm run setup-auth
npm run generate
npx playwright test --project=chromium --workers=1 --reporter=line
```

Go and repository guards:

```sh
go test ./... -count=1 -p 1 -timeout=5m
go vet ./...
go run ./tools/actionguard
go run ./tools/moduleguard -root modules -allow modules/api -dependencies modules/dependencies.json
go run ./tools/migrationguard -root migrations
go run ./tools/importguard -root examples/client-edition
go run ./tools/importguard -root examples/news-edition
go run ./tools/importguard -root examples/construction-edition
```

## Results

- Because the initial direct run was performed without preparing the fixture, generated Admin smoke
  tests redirected to the login page. This was not a product defect, but an omission in the test
  setup. Tests explicitly require a valid admin session.
- With the correct isolated server and `setup-auth.mjs`, the full Chromium matrix passed **63/63**.
  Scope covers 15 viewport/responsive checks, keyboard/focus and naming,
  admin role/CSRF, invite/password/MFA, schema-driven dynamic collection CRUD,
  generated CRUD, rich-text/relation/media, draft/publish, critical customer
  journeys, and public/Admin smoke tests generated from the manifest.
- On the first full run, the generated Admin axe scan produced a single contrast warning;
  an isolated individual rerun and a full rerun with fixture-auth passed. Going forward, the E2E helper
  also includes the axe node failure summary in the assertion output.
  For this generated CRUD page, eight theme/mode combinations were also scanned via axe-core:
  light/dark for HydroGo, shadcn, Nova, and Aurora;
  **8/8 passed.** The audit discovered avatar contrast in dark HydroGo mode and
  shadcn/Nova themes overriding the fixed avatar palette. Palette colors
  were made independent of theme switches; the full Axe scan in the matrix now runs
  across all eight combinations. Theme-based WCAG acceptance for other Admin pages and all public/generated
  types remains open.
- In a follow-up, the generated CRUD list view was scanned separately with axe-core across eight
  family/mode combinations: light and dark for HydroGo, shadcn, Nova, and Aurora.
  **8/8 passed.** The audit identified white avatar text contrast against the HydroGo accent color
  in dark mode, and shadcn/Nova themes overriding the avatar palette.
  Avatar palette colors were decoupled from the theme; tests now verify every theme/mode.
  This submatrix does not claim that all Admin routes or public/generated page types have been scanned in every theme.
- In the 2026-10-07 follow-up, the generated CRUD create form was also scanned with axe-core across the same four theme families
  and light/dark mode combinations: **8/8 passed**.
  Form inspection is also included in the responsive Chromium flow; other Admin
  routes and public pages across all theme/mode combinations remain out of scope.
- In a security follow-up, media list access was tested using editor/viewer sessions;
  when an administrator revoked the `media.manage` permission from the editor, new sessions received a 403
  on the `/admin/media` route, and upon revoking the permission, existing sessions were invalidated.
  The standalone Chromium role flow passed **1/1**. Media handlers now use the same
  `auth.Can` check; permission denial returns a 403, distinct from server component absence.
- Admin quick search now restricts pages/news, media, members, and site settings
  results via `content.read`, `media.manage`, `members.manage`, and
  `site.manage`, respectively. In a real editor session with a `content.read` deny
  override, page/news search results are absent; two role/permission tests in the same admin-management
  Chromium flow passed. The complete resource-based Go role and override table can be found in `internal/search/admin_http_test.go`.

In the 2026-10-07 follow-up run, `admin-management.spec.ts` added a test reusing the existing admin storage
state to verify that 15 management routes return 200 on both 390px and 768px viewports
without horizontal overflow. As a result, the full Chromium matrix passed **64/64** (2.9 minutes).
The Admin desktop visual golden run passed 7/7. Tablet/mobile visual projects could not be launched
on this machine due to missing WebKit browser launch dependencies; responsive acceptance at the same viewport
dimensions was executed via Chromium. This result should not be construed as physical device
or WebKit visual parity testing.

The Admin and MFA group also passed 6/6 across `custom/admin-management.spec.ts`,
`custom/admin-password-reset.spec.ts`, and `custom/admin-mfa-settings.spec.ts`,
including the new responsive acceptance test. The suite independently repeats the same authentic
login/challenge and mutation flows.

In mobile usability follow-up, target heights of at least 44px were established for the Admin Studio menu and common
form/card/action buttons at 390px and 768px widths, with tooltip target sizes defined as at least 24x24px.
Rich-text toolbar controls are exempt from this button rule to maintain compact layout density.
`admin-management.spec.ts` traverses 15 management routes across two viewports to measure target
sizes; the standalone test passed **1/1**, all management/role/invite flows in the same file passed
**4/4**, and `go test ./internal/ui -count=1` passed. This does not replace desktop
golden or WebKit/physical device verification.

The same mobile test opens the hamburger menu at 390px and verifies `aria-expanded` and open
state; it verifies closing via Escape, immediate reopening, and selecting a menu
link navigating to the target route and closing the menu. The immediate reopening attempt revealed
that an existing 250ms click lock swallowed user input; the redundant lock was removed. With the updated
menu flow, the standalone test passed **1/1**, and the full admin-management file passed **4/4**.

In Phase 1 progression, a new `milestone` Blueprint resource was added to Project Edition.
The same generated Admin registration helper added this resource to the menu without requiring an explicit
ID definition; the API route and approved migration were explicitly wired by the Edition.
Browser testing executed create/edit/delete and API readback on the new collection, scanned the list and form with axe,
and verified 390px form overflow. The dark/light four-theme-family matrix passed again on the existing Project list.
On the initial new resource run, collection description contrast measured 3.7:1;
the shared `.dashboard-card-head .muted` style was migrated to the semantic
`--ink-secondary` token, and the entire Chromium matrix passed once again.
Resource and DB contract verification is documented in `blueprint-derived-admin-activation-2026-10-07.md`.

## Follow-up — Theme WCAG Matrix Across All Admin Routes

On 2026-10-07, using a single authentic Edition admin session, 14 management routes, four
Studio themes (HydroGo, shadcn, Nova, Aurora), and light/dark modes were evaluated with axe-core
WCAG 2.1 A/AA scans: **112/112 combinations passed**. Low-contrast status/permission badges, media text, and
secondary text hues flagged in earlier runs were resolved. Test command and exact scope:
`docs/validation/admin-theme-accessibility-2026-10-07.md`.
This result should not be interpreted as WebKit/device visual testing or theme
coverage for all public/generated types.

## Linux WebKit Tablet/Mobile Visual Goldens

In the 2026-10-07 follow-up validation, the `visual-tablet` and `visual-mobile` projects utilizing
Playwright's WebKit default were executed in the official `mcr.microsoft.com/playwright:v1.63.0-noble` container
matching `@playwright/test` 1.63.0. An isolated seeded SQLite Edition, temporary admin account, and corporate
theme were utilized. **14/14** Linux full-page screenshot baselines were generated for Home, news, admin login,
dashboard, page editor, news editor, and media library; rerunning with the same session/fixture without
`--update-snapshots` also passed **14/14**. In a live 390x664 WebKit viewport, the news editor's sticky save
action measured 44px at the bottom of the screen. This golden suite covers seven surfaces; coverage of all
management routes, all field variants of generated collections, or physical iOS devices is not claimed.

To rerun the scope under identical conditions:

```sh
HYDRO_VISUAL=1 npx playwright test visual.spec.ts --project=visual-tablet --project=visual-mobile --workers=1
```
