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

The collection hub, schema-driven create/list screen, and edit form had no
automated axe-core WCAG 2.1 A/AA violations in the test. At 390px viewport
width, the document has no horizontal overflow. The schema-driven records
screen was visually inspected at desktop size; it uses the Admin Studio shell,
navigation, form and table styling.

Every registered collections HTML/API handler is also invoked with authenticated
editor and viewer contexts; both roles are denied with 403. All routes are
asserted to remain protected. This route-handler role matrix complements the
browser test's real administrator journey.

SQLite and live PostgreSQL storage contracts cover schema/record CRUD, required
and unique fields, duplicate-write rollback, relation existence and deletion
protection, version increments, and tenant isolation. PostgreSQL's unique-value
race test starts eight writers together and requires exactly one persisted
record. The live test uses a temporary schema and drops it during cleanup.
SQLite migration filtering verifies the four collection tables are absent when
the module is disabled and present when enabled. A runtime startup test checks
that `/admin/collections` is missing with the module disabled and remains behind
authentication when enabled.

## Commands

```sh
go test ./internal/ui ./internal/adminstudio ./internal/collections/... ./cmd/hydro -count=1
cd e2e && npx playwright test custom/dynamic-collections.spec.ts --project=chromium --workers=1 --reporter=line
```

The Go contract was also run against live PostgreSQL with
`HYDRO_TEST_POSTGRES_URL` set to an isolated test connection; PostgreSQL
collection contract passed. The targeted database, transport, runtime route,
and browser tests passed; the browser test passed 1/1. Screenshots and traces are
stored in the ignored `e2e/artifacts/` directory and are not committed.

This validates the runtime dynamic-collections module. It does not claim that
every Blueprint-generated resource becomes active without Edition composition;
generated route, authorization, and migration activation remain explicit by
design under the existing ADRs.

## Blueprint sync follow-up — 2026-10-08

The same Chromium journey now creates a generated `sync_articles` Blueprint,
runs `hydro collections sync --confirm-admin-crud`, and verifies that the
imported schema appears in the existing Admin Collections panel. The imported
resource's record create, update, and delete workflow passed in the real
temporary Edition. Command:

```sh
cd e2e && npx playwright test custom/dynamic-collections.spec.ts --project chromium --output /tmp/hydro-e2e-blueprint-sync
```

Result: 1/1 Chromium test passed. The normal `e2e/artifacts` output directory
was not writable in this workspace, so Playwright results were directed to a
temporary output directory.

## Admin form layout follow-up — 2026-10-08

Rendered two-field record forms now keep the fields in equal columns; a single
field still spans the row, while rich-text fields stay full width. The
Playwright journey checks the two rendered input widths differ by no more than
one pixel and asserts the Admin card styles are active under CSP. At 390px,
the collection form stacks cleanly; the mobile navigation opens with its shadow
and closes with Escape, after which the shadow no longer covers page content.
The same test confirms there is no horizontal document overflow. The layout
check is implemented by a scoped record-form class and included in the Admin
style hash catalog.

## Theme and color-mode accessibility follow-up — 2026-10-08

The generated Collections hub, schema-driven records/create surface, and
record-edit form are each checked with axe-core WCAG 2.1 A/AA under all four
Studio theme families (`hydrogo`, `shadcn`, `nova`, `aurora`) in light and dark
modes. Each rendered route returns 200 and exposes the selected theme and mode.
The complete CRUD journey, including this 24-screen accessibility matrix,
passed in Chromium (1/1 test).
