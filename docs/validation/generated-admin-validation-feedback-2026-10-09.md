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

Playwright `--list` discovered the generated CRUD spec (1/1), including its
real-browser 422 scenario. The actual browser run stopped in setup at
`spawnSync go EPERM`, before Edition migration, server startup, or Chromium
page creation. Thus no browser or axe result is claimed. The standalone
moduleguard, migrationguard (110 migrations), all three Edition importguards,
assetguard, and actionguard also passed in the follow-up verification. The
media visual-quality detector's six shell findings are logged above rather
than presented as a clean detector run. `HYDRO_TEST_POSTGRES_URL` was unset, so
there is no new PostgreSQL run. These external gates remain open. The UI
contract is recorded in [ADR-0364](../architecture/0364-generated-admin-validation-feedback.md).
