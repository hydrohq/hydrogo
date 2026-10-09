# News Admin list role flows — 2026-10-09

## Scope

Added a direct HTTP-handler rendering test for the News collection list with a
fixed repository response. It exercises default Editor and Viewer read access,
role-specific row actions, the bounded list request, and the filtered empty
state.

## Verified

- The Editor and Viewer both see the saved draft row.
- The Editor sees “Düzenle”; the Viewer sees “Görüntüle” and no edit action in
  the rendered list table.
- The list repository receives a limit of 100.
- Filtering the draft out with `status=published` renders an empty-state row
  and accessible `role="status"` feedback.
- `go test ./modules/news/... -count=1 -p 1 -timeout=5m`,
  `go vet ./modules/news/...`, and the focused transport coverage run passed.
- News transport statement coverage changed from 38.6% to 42.3% with this test.

## Limits

The test uses a repository fake. It does not prove database persistence or
PostgreSQL behavior; those remain separate contract tests. This direct HTML
test is not browser, axe, keyboard, responsive, or visual-golden evidence.
