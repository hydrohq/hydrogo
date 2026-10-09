# Generated form interface locale — 2026-10-10

Generated record forms now use the selected Admin interface locale for create
and edit headings, the save/save-draft action, the one-to-one relation
placeholder, and the read-only detail return link. Turkish remains the
default catalog; English strings come from the same catalog mechanism as the
generated list and validation UI. Collection and field labels remain Edition
content and are not translated by this change.

The `TestGeneratedAdminRendersListAndCreateForm` HTTP test renders the create
form in English and confirms the English heading and save action appear while
the Turkish heading/save controls do not. The direct typed block-editor test
continues to cover generated field rendering.

Verification passed:

```sh
GOCACHE=/tmp/hydrogo-admin-locale-cache go test ./internal/adminstudio ./internal/i18n -count=1
GOCACHE=/tmp/hydrogo-admin-locale-cache go vet ./internal/adminstudio ./internal/i18n
```

This closes only the tested form-shell labels. Other generated workflow,
revision, delete-confirmation, conflict, and field-widget copy still needs a
locale audit. This is a translation/UI change within the existing Admin
Studio contract; it adds no route, migration, policy, or persistence behavior.
