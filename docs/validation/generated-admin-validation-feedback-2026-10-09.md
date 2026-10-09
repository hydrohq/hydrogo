# Generated Admin validation feedback — 2026-10-09

Generated CRUD now handles a structured Edition API `422 validation_failed`
response by returning the form with an accessible validation summary and links
to known editable fields. Turkish and English field guidance distinguishes
required fields from other validation errors. Unknown field names and API
message text are ignored. Create preserves submitted values; update reloads the
current resource to restore untouched fields before overlaying submitted values.
Links target standard controls, rich-text editor, block-editor mount, media
field, and array editor wrappers.

The HTTP tests verify status 422, locale copy, field links and focus targets,
preserved values, update reload, ignored unknown fields, and no API message
leakage. Component tests verify escaped accessible summary markup and stable
widget IDs. The following checks passed:

```sh
GOPROXY=off go test ./internal/adminstudio ./internal/ui/components ./internal/i18n -count=1
GOCACHE="$PWD/.cache/go-build" GOTMPDIR=/tmp TMPDIR=/tmp GOPROXY=off \
  go test ./... -count=1 -p 1 -timeout=8m
GOCACHE="$PWD/.cache/go-build" GOTMPDIR=/tmp TMPDIR=/tmp GOPROXY=off \
  go vet ./...
GOCACHE="$PWD/.cache/go-build" GOTMPDIR=/tmp TMPDIR=/tmp GOPROXY=off \
  go run ./tools/moduleguard -root modules -allow modules/api -dependencies modules/dependencies.json
GOCACHE="$PWD/.cache/go-build" GOTMPDIR=/tmp TMPDIR=/tmp GOPROXY=off \
  go run ./tools/migrationguard -root migrations
GOCACHE="$PWD/.cache/go-build" GOTMPDIR=/tmp TMPDIR=/tmp GOPROXY=off \
  go run ./tools/importguard -root examples/client-edition
GOCACHE="$PWD/.cache/go-build" GOTMPDIR=/tmp TMPDIR=/tmp GOPROXY=off \
  go run ./tools/importguard -root examples/news-edition
GOCACHE="$PWD/.cache/go-build" GOTMPDIR=/tmp TMPDIR=/tmp GOPROXY=off \
  go run ./tools/importguard -root examples/construction-edition
GOCACHE="$PWD/.cache/go-build" GOTMPDIR=/tmp TMPDIR=/tmp GOPROXY=off \
  go run ./tools/assetguard -root .
GOCACHE="$PWD/.cache/go-build" GOTMPDIR=/tmp TMPDIR=/tmp GOPROXY=off \
  go run ./tools/actionguard
git diff --check
```

The Impeccable detector returned no finding on the generated validation
handler, typed summary component, or CRUD browser assertion. Its scan also
reported six warnings in the shared Admin shell: the rich-text blockquote
accent, neutral nested-navigation border, bundled Inter face, and existing
progress/timer width transitions. Those exact snippets are present unchanged
at `HEAD`; the warnings are outside this validation-feedback change. No
migration, database query, or schema behavior changed; SQLite/PostgreSQL
specific behavior therefore remains covered by the existing CRUD contracts,
not a new validation-specific DB test.

Playwright's full theme matrix now submits the invalid required field in each
of four Studio themes and both color modes, then runs axe on each resulting
422 page. The generated CRUD Chromium journey passed 1/1 on local SQLite,
including that matrix and full create/edit/delete behavior. A previous run had
failed because the default Go build cache was read-only, followed by loopback
listener denial; the current runner successfully started the Edition and opened
Chromium. The E2E Go command wrapper remains strict: it accepts `EPERM` only
when `spawnSync` reports exit status 0 and rejects the actual exit-1 read-only
cache failure.

The standalone moduleguard, migrationguard (110 migrations), all three Edition
importguards, assetguard, and actionguard also passed in the follow-up
verification. The media visual-quality detector's six shell findings are
logged above rather than presented as a clean detector run. The UI contract is
recorded in [ADR-0364](../architecture/0364-generated-admin-validation-feedback.md).

## Generated mutation CSRF route matrix follow-up

`TestGeneratedAdminAllMutationRoutesRejectInvalidCSRFBeforePersistence` now
exercises nine enabled generated mutation routes in a single Edition fixture:
create, update, delete, workflow transition, revision restore, locale save,
locale copy, draft save, and draft discard. Each route is now tested with both
a missing token and a mismatched token; both return `403`. The test asserts
that neither the composed resource API nor any Edition-owned persistence/
policy callback is reached. `go test ./internal/adminstudio -run
'^TestGeneratedAdminAllMutationRoutesRejectInvalidCSRFBeforePersistence$'
-count=1`, the full package test, and `go vet ./internal/adminstudio` passed.
Existing tests continue to cover valid-token success, capability denial,
stale-version conflict, and persisted editorial state. This closes the
generated-resource missing/mismatched CSRF handler slice only; built-in Admin
routes and browser role/session behavior remain separately scoped.

The generated CRUD Playwright fixture also supports an externally hosted,
loopback-only Edition server via `HYDRO_E2E_EXTERNAL_SERVER=1`,
`HYDRO_E2E_BASE_URL`, and `HYDRO_E2E_DATABASE_DSN`. It checks `/healthz` and
leaves the external server/database lifecycle to its caller. This makes the
browser scenario usable when the test runner cannot bind sockets; the external
mode itself has not run in this environment because no reachable Edition
server is available.

## Light-theme error contrast follow-up

Source-level WCAG calculations found `.studio-alert` used the palette's normal
red text on pale red backgrounds, yielding 4.04:1 in HydroGo, 4.28:1 in shadcn,
and 4.41:1 in Nova/Aurora. `AdminCSSOverrides` now uses `#991b1b` for light-mode
alert text. Contrast is 7.28:1–7.60:1 across those four background tokens;
dark mode retains each theme's existing red text. A Go regression test reads
the four source palette tokens and checks the 4.5:1 threshold. The focused
accessibility tests and full `internal/ui` + `internal/adminstudio` packages,
their `go vet`, and `git diff --check` passed. The existing eight-state browser
axe assertion still could not execute because the runner cannot bind the
Edition's loopback server.

## Search/filter empty state — 2026-10-09

Generated resource lists now distinguish an actually empty collection from a
search/filter with no matches, and choose Turkish or English using the Admin
interface locale rather than the content locale. The empty-state text gives a
clear recovery hint. Go tests cover both locales, empty/search/filter states,
and the accessible `role="status"` announcement. The existing Chromium CRUD
journey now expects the no-match message. A registered generated-handler test
also confirms the Admin language cookie selects English even when a content
locale is not involved. Generated Admin/i18n Go tests and vet passed;
Impeccable's detector returned no findings. Playwright is now available for
test discovery, but browser execution is still open because this environment
denies loopback listener creation. No browser pass is claimed.

## Native Blueprint required-field browser journey — 2026-10-09

The Project Edition's native generated `milestone` CRUD Playwright journey now
submits a whitespace-only required title before the successful create. It
asserts the 422 response, localized accessible summary, field-level feedback,
preservation of the submitted value, and axe WCAG 2.1 A/AA results before
continuing to create the valid record. The full Chromium CRUD journey passed
1/1 on the local SQLite Edition; it includes CRUD, four themes in light/dark
modes, mobile/tablet overflow checks, axe scans, keyboard access, and the
required-field recovery flow. PostgreSQL and deployed Edition browser coverage
remain open.

```sh
npx playwright test --project=chromium --workers=1 custom/generated-admin-crud.spec.ts \
  --reporter=line --trace=off
```
