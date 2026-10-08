# Generated block editor validation — 2026-10-08

## Browser contract

Added a `FieldBlocks` column to the generated article test fixture and served
the official Editor.js 2.31.6 UMD bundle from the Edition origin. Its Apache
2.0 license is included beside the test asset. Chromium confirmed that the
collection list does not fetch the editor asset, the create form loads it, an
empty field starts with an editable paragraph, a second paragraph can be added,
and the API persists and restores the structured JSON.

This run exposed two shared-loader gaps: Editor.js does not make an editable
paragraph from an explicit empty `blocks` array, and its contenteditable
paragraphs need textbox roles and accessible names. The loader now seeds one
empty paragraph only for an explicitly empty array and labels editable blocks
when Editor.js creates them. DOM observation handles block additions without
scanning every block on each keystroke.

The final full generated editorial Chromium file passed 7/7 with the current
loader on SQLite and PostgreSQL 18. The block journey includes an axe WCAG 2.1
A/AA scan on both backends. The current run also checks the 390px viewport for
horizontal overflow and captures the mobile editor form. The fixture schema is
synthetic; it is not a Project Edition production migration. The upstream
bundle verifies actual Editor.js compatibility for the default Paragraph tool.
It does not certify a Project Edition deployment bundle, additional Editor.js
tools, or an Edition preview renderer.

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

The test asset is `@editorjs/editorjs` 2.31.6, `dist/editorjs.umd.js`, from the
official [Editor.js repository](https://github.com/codex-team/editor.js/).
