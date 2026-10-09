# Pages Admin list role flows — 2026-10-09

## Scope

Added direct HTTP-handler rendering coverage for the Pages collection list.
The test provides a fixed repository response and checks the visible list table
for an Editor and a Viewer, plus the accessible empty state after filtering a
draft out of the published view.

## Verified

- The Editor sees the saved page row, the edit action, and the permitted CSV /
  Excel import action.
- The Viewer sees the same row with a read-only “Görüntüle” action; the edit
  and import controls are absent from the rendered route output.
- A status filter that removes all rows produces the list's empty-state row and
  an accessible `role="status"` message.
- The list repository is called with the fixed 100-row bound.
- `go test ./modules/pages/transport -count=1 -p 1 -timeout=5m` passed;
  `go vet ./modules/pages/transport` passed.
- Package statement coverage changed from 39.5% to 42.1% with this test.

## Limits

The test uses a repository fake and does not establish SQLite/PostgreSQL list
query behavior or record persistence; those remain covered separately by
repository contracts. It is an HTML handler test, not a Chromium, keyboard, axe,
responsive, or visual-golden result. Those wider Admin UI gates remain open.
