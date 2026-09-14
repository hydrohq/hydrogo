# R5 validation: release asset hygiene

## Local evidence

- `go test ./tools/assetguard` verifies referenced and unreferenced runtime
  vendor assets, plus a missing-vendor-root case.
- `scripts/verify-local.ps1` invokes `assetguard` before build and release
  smoke checks.
- The guard maps embedded `static/vendor/...` paths to their public `/vendor/`
  route and passes against the current repository assets.
- Current HydroGo vendor JS/CSS assets are referenced by the editor/admin
  runtime surfaces.

## Remaining gate

Dynamic browser loading and final production image size still require the
browser/device and deployment image gates. The guard intentionally does not
delete files automatically.
