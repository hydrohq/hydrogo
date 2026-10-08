# Native Project Edition block editor validation — 2026-10-08

## Coverage

The Project Edition now has a Blueprint-generated `content_block` resource
with a required title and Editor.js `TEXT` body. The generated resource joins
the Edition's API and Admin registry automatically. Its separate migration
keeps Article's existing Quill data unchanged.

The Edition serves its pinned Editor.js 2.31.6 bundle itself. When a preview
is configured, the hash-authorized Admin shell loader also loads the shared
same-origin HTMX runtime on demand and processes the form. Preview sends only
the body and CSRF token, validates the bounded Paragraph document, sanitizes
text, and responds with private no-store HTML. Mutations use the same bounded
validation and sanitization before generated persistence.

## Verification results

- `TMPDIR=/var/tmp go test ./... -count=1` passed across the repository.
- `go vet ./...` passed.
- `go run ./tools/moduleguard -root modules -allow modules/api -dependencies modules/dependencies.json` passed.
- `go run ./tools/migrationguard -root migrations` passed (110 migrations).
- `go run ./tools/assetguard -root .` passed.
- The CI Edition import guards passed for Client, News, and Construction Editions.
- `go test ./examples/project-edition/custom -run '^TestPostgresEditionGeneratedAdminCRUD$' -count=1 -timeout=2m` passed against a disposable PostgreSQL 18 instance. This covers the new PostgreSQL migration and sanitized API persistence.
- The real Project Edition Chromium CRUD journey passed on SQLite and PostgreSQL 18. It checks the collection appears automatically, the block form loads, the live preview returns 200 and renders sanitized text, create/read/update/delete work, and the 390px form has no horizontal overflow or axe violations.
- The block editor create and edit forms passed an axe WCAG 2.1 A/AA scan in all eight combinations of HydroGo, shadcn, Nova, and Aurora with light and dark modes. The title and editor remained visible, and the saved paragraph survived each edit-page reload.
- The saved paragraph uses the shared Admin `--ink` theme token for text and caret; this rule is authorized by the shell's CSP style hash. The test waits for Editor.js's short block fade-in before running axe so it checks the settled UI rather than a transient opacity value.
- The browser flow confirms that collection lists and forms without a preview-enabled block field do not load HTMX; the runtime is present on the configured block form only.
- `git diff --check` passed.

The browser command was:

```sh
cd e2e
npx playwright test custom/generated-admin-crud.spec.ts --project=chromium \
  --workers=1 --grep 'finds a generated collection' --reporter=line \
  --output=/tmp/hydrogo-e2e-block-editor
```

For PostgreSQL, the same command was run with `HYDRO_TEST_POSTGRES_URL` pointing
to a disposable PostgreSQL 18 database. The test migration and browser data
were isolated to that disposable database, which was removed after the run.

## Known scope

The example deliberately accepts only Paragraph blocks. Adding Editor.js tools
requires matching schema validation, sanitization, rendering, and browser
coverage. The Project Edition's demo authorization remains loopback-only and
is not production identity or tenant policy.
