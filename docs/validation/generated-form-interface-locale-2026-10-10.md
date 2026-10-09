# Generated form interface locale — 2026-10-10

Generated record forms and editorial actions now use the selected Admin
interface locale for create/edit headings, save/save-draft, relation
placeholder, read-only detail return, record/draft delete confirmations,
workflow transitions, revision history/detail/restore actions, and stale edit
conflict feedback. Revision state labels use the existing localized status
catalog. Turkish remains the default catalog; English strings use the same
catalog mechanism as the generated list and validation UI. Collection and
field labels remain Edition content and are not translated by this change.

The `TestGeneratedAdminRendersListAndCreateForm` HTTP test renders the create
form in English and confirms the English heading and save action appear while
the Turkish heading/save controls do not. The direct generated field test
checks the English relation placeholder, workflow action test checks all four
standard transitions in English and Turkish, and the revision HTTP handler
test checks translated detail/restore/back controls with escaped snapshot
content. The stale-conflict handler test verifies the Turkish catalog copy.

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
