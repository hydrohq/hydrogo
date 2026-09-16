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
a real Client Edition and external authentication remains open.
