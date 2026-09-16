# R5 validation: headless starter TypeScript acceptance

Date: 2026-09-16

From `examples/headless-starter`:

- `npm run check` passes (`tsc --noEmit`).
- `npm run build` passes and emits the starter distribution.
- `npm run demo` reaches the configured HydroGo endpoint but cannot complete
  while no local server is listening on `127.0.0.1:8090` (`ECONNREFUSED`).

The last result is an environment precondition, not a starter compile error;
run the demo with `hydro serve` available. Browser acceptance against a real
Client Edition remains open.
