# Generated draft inbox localization validation — 2026-10-09

The generated draft inbox passes a localized list caption and action-column
label to the shared Admin Studio list renderer. Existing list views retain
their Turkish defaults when those optional values are omitted. The English
handler assertion now checks the screen-reader caption and responsive action
cell label in addition to visible headers and row status.

Validation passed:

```sh
GOCACHE=/tmp/hydro-cover-cache go test ./internal/adminstudio ./internal/i18n -count=1 -p 1
GOCACHE=/tmp/hydro-cover-cache go vet ./internal/adminstudio ./internal/i18n
/home/arda/.agents/skills/impeccable/scripts/impeccable detect --json \
  internal/adminstudio/generated_http.go internal/adminstudio/generated_http_test.go \
  internal/adminstudio/list.go internal/i18n/catalogs/en.json internal/i18n/catalogs/tr.json
```

The detector returned no findings. This is server-rendered handler evidence; the
Playwright localhost listener remains blocked in this managed environment as
recorded in `generated-draft-inbox-ux-2026-10-09.md`.

The shared-renderer change also passed the full root SQLite Go suite:
`GOCACHE=/tmp/hydro-cover-cache go test ./... -count=1 -p 1 -timeout=8m`.
