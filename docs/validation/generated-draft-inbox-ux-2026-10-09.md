# Generated draft inbox UX validation — 2026-10-09

The generated draft inbox now uses the Admin Studio's existing typed
`RenderList` table and status-pill rendering. It shows title, translated
workflow state, version, last-updated time and a direct open action; collection
labels remain Edition-owned. The table's wrapper uses the existing responsive
list CSS. UI copy for Turkish and English is in the Studio catalog, with a
localized accessible empty-state message. Localized collections keep the active
content locale when opening a draft.

The handler acceptance test verifies the Turkish and English table headings,
status pill, detail URL, responsive wrapper, and accessible empty state. It
also runs the existing draft isolation and compare-and-swap flow assertions in
the same test. `go test ./internal/adminstudio -count=1 -p 1 -timeout=5m`,
`go vet ./internal/adminstudio`, and the Impeccable detector passed.

This is handler-rendered HTML evidence. It does not claim browser screenshot,
axe, keyboard, theme, cross-viewport, or persistence coverage. Those remain
open acceptance gates in the product audit plan.

After the focused checks, the full root SQLite Go suite also passed:
`GOCACHE=/tmp/hydro-cover-cache go test ./... -count=1 -p 1 -timeout=8m`.
