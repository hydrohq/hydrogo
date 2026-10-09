# Generated form interface locale — 2026-10-10

Generated record forms and editorial actions now use the selected Admin
interface locale for create/edit headings, save/save-draft, relation
placeholder, read-only detail return, record/draft delete confirmations,
workflow transitions, revision history/detail/restore actions, and stale edit
conflict feedback. The content-locale navigation's accessible name and
cross-locale copy heading/action also follow the independent Admin UI locale.
Revision state labels use the existing localized status catalog. Turkish
remains the default catalog; English strings use the same catalog mechanism as
the generated list and validation UI. Collection and field labels remain
Edition content and are not translated by this change.

The `TestGeneratedAdminRendersListAndCreateForm` HTTP test renders the create
form in English and confirms the English heading and save action appear while
the Turkish heading/save controls do not. The direct generated field test
checks the English relation placeholder, workflow action test checks all four
standard transitions in English and Turkish, and the revision HTTP handler
test checks translated detail/restore/back controls with escaped snapshot
content. The locale handler test renders the content-locale navigation and
copy actions in both Turkish and English. The stale-conflict handler test
verifies the Turkish catalog copy.

Verification passed:

```sh
GOCACHE=/tmp/hydrogo-admin-locale-cache go test ./internal/adminstudio ./internal/i18n -count=1
GOCACHE=/tmp/hydrogo-admin-locale-cache go vet ./internal/adminstudio ./internal/i18n
```

This closes interface copy for the listed form and editorial controls, not the
full locale audit. Edition-provided field/widget labels and every browser
locale journey still need review. This is a translation/UI change within the
existing Admin Studio contract; it adds no route, migration, policy, or
persistence behavior.

## Chromium follow-up

The native generated CRUD Playwright journey now opens the real Project Edition
create form with the English Admin locale and asserts the translated heading
and submit action. Playwright discovery lists the scenario (1/1), but execution
in this managed workspace stops before server startup because binding the
loopback listener fails with `listen EPERM`. Therefore the browser locale
assertion is present but not passed here; the earlier Go HTTP-render assertion
remains the successful runtime-level evidence.

```sh
cd e2e
TMPDIR=/tmp GOCACHE=/tmp/hydro-cover-cache npm exec playwright test \
  custom/generated-admin-crud.spec.ts -- --project=chromium --workers=1 \
  --reporter=line --trace=off
npm exec playwright test custom/generated-admin-crud.spec.ts -- \
  --project=chromium --workers=1 --list
```

The discovery command passed with one scenario. The browser command did not
reach its assertions and is not counted as a passing E2E test.


## Generated widget controls follow the interface locale

Generated `Array`, `Media`, `RichText`, and `Block` fields now receive their control labels from the active Admin interface catalog. This covers add-item labels, media selection/clear controls, rich-text toolbar accessible names, and the block preview placeholder. Edition field labels and the selectable content locale remain separate from the interface locale. Shared typed components retain their existing Turkish defaults when called without localized props.

The generated form HTTP test renders all four widgets in English and asserts their control copy and absence of the corresponding Turkish strings. Component tests cover supplied English copy for array, media, rich-text, and block controls. Verification passed:

```sh
GOCACHE=/tmp/hydro-admin-gocache go test -count=1 -p 1 -timeout 90s ./internal/ui/components ./internal/adminstudio ./internal/i18n
GOCACHE=/tmp/hydro-admin-gocache go vet ./internal/ui/components ./internal/adminstudio ./internal/i18n
/home/arda/.agents/skills/impeccable/scripts/impeccable detect --json internal/adminstudio/generated_http.go internal/ui/components
git diff --check
```

All three Go packages passed tests and vet; the UI detector returned no findings. This is HTTP/component-level locale evidence, not browser proof. The Playwright listener restriction above still prevents a real browser locale journey in this workspace.
