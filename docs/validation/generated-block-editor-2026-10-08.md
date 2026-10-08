# Generated block editor validation — 2026-10-08

## Browser contract

Added a `FieldBlocks` column to the generated article test fixture and served a
small Editor.js-compatible test adapter from the Edition origin. Chromium
confirmed that the collection list does not fetch the editor asset, the create
form loads it, edits become structured JSON in the canonical hidden field, the
API persists the value, and the edit view restores both JSON and editable text.

The focused block journey passed with the fixture's in-memory/SQLite-style
store and a temporary PostgreSQL 18 schema (1/1 each). After adding the block
field, the full generated editorial Chromium file also passed 7/7 on both
backends. The focused journey also passed the axe WCAG 2.1 A/AA scan on both
backends. The fixture schema is synthetic; it is not a Project Edition
production migration. The adapter proves the shared loader and Editor.js
constructor contract. It does not certify a selected upstream Editor.js
distribution or any Edition's preview renderer.

## Commands

```sh
cd e2e
npx playwright test custom/generated-richtext.spec.ts --project=chromium \
  --workers=1 --grep 'generated blocks' --reporter=line \
  --output=/tmp/hydro-generated-block-editor
HYDRO_TEST_POSTGRES_URL='postgres://<user>:<password>@127.0.0.1:<port>/<database>?sslmode=disable' \
  npx playwright test custom/generated-richtext.spec.ts --project=chromium \
  --workers=1 --grep 'generated blocks' --reporter=line \
  --output=/tmp/hydro-generated-block-editor-postgres
```

The PostgreSQL server was a disposable PostgreSQL 18 container using an
unprivileged application account and was removed after the run.
