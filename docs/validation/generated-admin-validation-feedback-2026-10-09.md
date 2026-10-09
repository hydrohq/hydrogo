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
GOPROXY=off go test ./... -count=1 -p 1 -timeout=8m
GOPROXY=off go vet ./...
go run ./tools/moduleguard -root modules -allow modules/api -dependencies modules/dependencies.json
go run ./tools/migrationguard -root migrations
go run ./tools/importguard -root examples/client-edition
go run ./tools/importguard -root examples/news-edition
go run ./tools/importguard -root examples/construction-edition
git diff --check
```

The Impeccable detector returned no findings for the changed generated Admin
and UI component files. No migration, database query, or schema behavior
changed; SQLite/PostgreSQL-specific behavior therefore remains covered by the
existing CRUD contracts, not a new validation-specific DB test.

Playwright `--list` compiled the generated CRUD spec, including its real-browser
422 scenario. The actual browser run stopped in setup at `spawnSync go EPERM`,
before Edition migration, server startup, or Chromium page creation. Thus no
browser or axe result is claimed. `HYDRO_TEST_POSTGRES_URL` was unset, so there
is no new PostgreSQL run. These external gates remain open. The UI contract is
recorded in [ADR-0364](../architecture/0364-generated-admin-validation-feedback.md).
