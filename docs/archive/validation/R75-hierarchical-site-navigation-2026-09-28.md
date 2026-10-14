# R75 — Hierarchical Site navigation

The Site module now stores localized parent/target fields through migration
`202609280023_navigation_tree`. Existing rows default to root-level, same-tab
links. The service bounds a tree to 500 rows and depth 8, rejects duplicate
IDs, missing parents, cycles, invalid targets, and unsafe URLs, then replaces
one locale's menu in one transaction.

`/admin/site/navigation/editor` provides native drag-and-drop ordering,
keyboard move controls, parent selectors, visibility and target options, and
up to 200 same-locale published page suggestions. The public navigation
projection omits hidden nodes and their descendants. The corporate theme gets
the localized nested tree in its `navigation` slot and renders child lists with
escaped labels and URLs; `_blank` links add `noopener noreferrer`. The page
layout falls back to its existing links when no menu exists.

SQLite repository/service and editor integration tests cover tree persistence,
hidden descendants, invalid trees, localized suggestions, and the rendered
theme slot. The repository dialect contract includes the new columns for both
SQLite and PostgreSQL. The PostgreSQL visibility/repository contract ran on a
temporary PostgreSQL 18.6 instance and passed.

Verification performed:

```text
go test ./extension/v1 ./modules/pages/transport ./modules/site/... ./internal/ui ./cmd/hydro ./migrations ./tests/contract
PASS
```

The migration adds two columns only when the Site module is enabled. It adds no
index, worker, cache, or service. Each public query is bounded to 500 items;
editor page suggestions stop after 200 matching published pages.
