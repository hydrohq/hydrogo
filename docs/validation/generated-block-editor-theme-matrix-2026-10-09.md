# Generated block editor theme matrix — 2026-10-09

## Change

The generated Editor.js form browser journey now checks the real editor in all
four Admin theme families (`hydrogo`, `shadcn`, `nova`, `aurora`) and both light
and dark modes. Each variant asserts the active theme attributes, a mounted
editable block, and axe WCAG 2.1 A/AA results before the CRUD round trip.

## Verification status

- Playwright discovered all seven tests in
  `custom/generated-richtext.spec.ts`.
- The run stopped in suite setup before any test assertion: the generated
  fixture could not bind its loopback server (`listen EPERM: operation not
  permitted 127.0.0.1`). The theme matrix result is therefore **unverified**.
- `git diff --check` passed.

## Open gate

Run the suite on a runner that permits loopback listeners:

```sh
cd e2e
npx playwright test custom/generated-richtext.spec.ts --project=chromium --workers=1
```

Record actual axe findings and screenshots before treating the generated block
editor's theme/accessibility coverage as complete. This remains in-process
browser validation, not customer deployment evidence.
