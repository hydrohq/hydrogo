# Dynamic Collections Admin Validation

**Date:** 2026-10-07  
**Scope:** Opt-in `collections` module, one Edition admin, SQLite, Chromium.

## Result

The dynamic collection workflow is reachable from the Admin sidebar when the
module is enabled. The link is absent when disabled, is restricted to the
admin role, and shows the current-page state on collection routes. The dashboard
uses a dedicated renderer, so its module flag is passed explicitly; normal
request-backed Admin pages use the shared support context.

The browser test starts the actual HydroGo server against a temporary SQLite
database and walks through schema creation, adding required and unique fields,
creating a record from the generated form, checking the collection API,
editing and deleting the record, and deleting the schema. A schema POST without
CSRF is rejected with 403. The flow uses the fixed collection storage model and
does not execute runtime DDL.

The collection hub had no automated axe-core WCAG 2.1 A/AA violations in the
test. At 390px viewport width, the document has no horizontal overflow. The
schema-driven records screen was visually inspected at desktop size; it uses
the Admin Studio shell, navigation, form and table styling.

## Commands

```sh
go test ./internal/ui ./internal/adminstudio ./internal/collections/... ./cmd/hydro -count=1
cd e2e && npx playwright test custom/dynamic-collections.spec.ts --project=chromium --workers=1 --reporter=line
```

Both commands passed. The browser test passed 1/1. Screenshots and traces are
stored in the ignored `e2e/artifacts/` directory and are not committed.

This validates the runtime dynamic-collections module. It does not claim that
every Blueprint-generated resource becomes active without Edition composition;
generated route, authorization, and migration activation remain explicit by
design under the existing ADRs.
