# R5 validation: headless starter TypeScript acceptance

Date: 2026-09-16

From `examples/headless-starter`:

- `npm run check` passes (`tsc --noEmit`).
- `npm run build` passes and emits the starter distribution.
- With a disposable SQLite database seeded by `hydro seed-demo` and a temporary
  `hydro serve --address 127.0.0.1:8090`, `npm run demo` completed and returned
  one page, three page summaries, eight news items, and eight products.
- The temporary server was stopped after the check; no local service was left
  running.

This proves the starter’s local API contract path. Browser acceptance against
a real external Client Edition and external authentication remains open.

Additionally, `scripts/run-e2e.ps1` completed locally with 47 Chromium tests
covering the generated frontend routes, keyboard focus, responsive behavior,
accessible names, Admin Studio journeys, media, and authenticated flows.
The disposable server and database were torn down by the script afterward.
