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
