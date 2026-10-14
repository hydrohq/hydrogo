# Release hygiene audit — 2026-09-15

## Scope

The release tree was checked for unused embedded vendor assets, demo-only
runtime exposure, duplicate route patterns, and obvious dead route/handler
registrations.

## Evidence

- `go test ./tools/assetguard ./tools/assetmanifest` passed; every embedded
  runtime asset has a known reference. HTMX and Quill are both used by the
  editor contract, so neither was removed.
- `go test ./internal/router ./cmd/hydro` passed; route construction and
  duplicate-pattern validation are covered by regression tests.
- Demo data is isolated behind `seed-demo --confirm-demo`, and the production
  image stage excludes demo/fixture files (existing release checks passed).
- A source route inventory (143 registrations/call sites) was reviewed. No
  registration could be removed safely without changing a documented public,
  admin, API, or module contract.

## Decision

No safe deletion candidate was found. The release-hygiene task is therefore
complete for the current revision; future route additions must keep the route
tests and asset guard green.
