# Generated collection list localization validation — 2026-10-09

Generated resource lists now localize the visually hidden collection caption,
the action-column heading and responsive action-cell label through the active
Admin UI locale. Edition-provided collection and field names remain intact.
The English handler test checks the empty-search state, caption, action heading
and a populated row's responsive action label. Existing Turkish list tests
continue to verify the fallback strings.

Validation passed:

```sh
GOCACHE=/tmp/hydro-cover-cache go test ./internal/adminstudio ./internal/i18n -count=1 -p 1
GOCACHE=/tmp/hydro-cover-cache go vet ./internal/adminstudio ./internal/i18n
GOCACHE=/tmp/hydro-cover-cache go test ./... -p 1 -timeout=8m
```

The Impeccable detector returned no findings. This is server-rendered and
cross-package Go evidence, not a browser/axe pass. Localhost listener access
remains blocked in the current environment, documented in
`generated-draft-inbox-ux-2026-10-09.md`.

The same active-locale resolver now supplies generated workflow status-pill
labels, including English Published and Draft labels; custom/unknown Edition
states retain their configured state string. The draft inbox and populated
collection list assertions cover both screens. The Admin Studio suite and
Impeccable detector passed after this addition.

After status-pill locale rendering was added, the full root Go suite passed again:
`GOCACHE=/tmp/hydro-cover-cache go test ./... -p 1 -timeout=8m`.

## Generated list toolbar locale coverage

The generated list toolbar and entry points now use the active UI locale for
Search, All, Apply, New record, Edit, Drafts, and pagination controls. Content
locale remains a separate value used for localized-resource query preservation.
The English handler acceptance now renders 26 records and checks Next on page
one and Previous on page two, while existing Turkish tests retain the Turkish
fallbacks.

Validation passed:

```sh
GOCACHE=/tmp/hydro-cover-cache go test ./internal/adminstudio -count=1 -p 1 -timeout=5m
GOCACHE=/tmp/hydro-cover-cache go vet ./internal/adminstudio ./internal/i18n
GOCACHE=/tmp/hydro-cover-cache go test ./... -p 1 -timeout=8m
/home/arda/.agents/skills/impeccable/scripts/impeccable detect --json \
  internal/adminstudio/generated_http.go internal/adminstudio/generated_http_test.go \
  internal/i18n/catalogs/en.json internal/i18n/catalogs/tr.json
```

The full root Go suite and detector passed. This does not replace the blocked
Playwright browser/axe check or the unavailable full PostgreSQL rerun.
