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
