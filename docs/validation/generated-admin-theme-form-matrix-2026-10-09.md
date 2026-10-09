# Generated Admin collection and editorial theme matrix — 2026-10-09

The generated Admin browser fixture now checks five rendered surfaces across
the four Studio theme families (`hydrogo`, `shadcn`, `nova`, `aurora`) in light
and dark mode:

- Article collection list
- Article create form, including rich text and blocks
- Story collection list
- Story draft inbox
- Localized Story edit form

Each of the 40 screen/theme combinations verifies the response, active theme
attributes, expected page content, and axe-core WCAG 2.1 A/AA rules. All passed
without axe violations. This extends the existing create-form/block-editor
theme check to collection and editorial surfaces; it is synthetic Edition
fixture evidence, not a claim about every product Admin route or a deployed
customer Edition.

The focused matrix passed, then the complete generated-richtext Chromium file
passed 9/9 against both fixture adapters:

```sh
cd e2e
npx playwright test custom/generated-richtext.spec.ts --project=chromium \
  --workers=1 --grep 'generated collection and editorial screens' \
  --reporter=line --trace=off
npx playwright test custom/generated-richtext.spec.ts --project=chromium \
  --workers=1 --reporter=line --trace=off
```

The full SQLite run passed 9/9 (1.1 minutes). The full PostgreSQL run passed
9/9 (1.1 minutes) with a disposable PostgreSQL 18.6 database and the externally
started fixture. The PostgreSQL editorial schema is fixture-owned and is not a
substitute for production Edition migration certification. The full suite also
covers Quill and Editor.js editing, create/edit/API round-trip, save retry,
stale-edit conflict recovery, relation/media, localized draft publication,
and draft review/publish/discard.
