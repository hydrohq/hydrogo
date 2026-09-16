# ADR-0127: Admin taxonomy editor primitive

- **Status:** Accepted
- **Date:** 2026-09-16
- **Related decisions:** [ADR-0120](0120-versioned-taxonomy-contract.md), [ADR-0029](0029-relation-field-rendering.md), [ADR-0081](0081-ui-accessibility-regression-contract.md)

## Decision

Admin Studio provides a provider-neutral `RenderTaxonomyEditor` primitive for
localized category/tag selection. It renders semantic fieldset/legend and
labelled checkboxes, escapes term IDs/labels, and sorts terms deterministically.
The owning module supplies terms, selected IDs, action handling, authorization,
CSRF validation and persistence; the primitive does not access a database or
create routes.

## Evidence and consequences

`internal/adminstudio/taxonomy_test.go` covers deterministic ordering and HTML
escaping. News exposes the protected `/admin/news/{id}/taxonomy` endpoint and
uses the primitive. When the database supports transactions, the endpoint wraps
replacement in a single commit/rollback boundary (with a regression probe in
`modules/news/transport/http_test.go`); term management and related-content UX
remain product work.
