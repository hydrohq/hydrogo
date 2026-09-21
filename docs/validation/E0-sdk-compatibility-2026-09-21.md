# E0 TypeScript SDK/API Compatibility Gate — 2026-09-21

`scripts/verify-sdk-compat.ps1` is the repeatable release gate for the
generated TypeScript surface. It runs generator/project/sdkdist Go contract
tests, creates a disposable generated resource, validates every generated
artifact with `hydro validate generator`, and compiles/runs the independent
`examples/headless-starter` consumer.

The external consumer remains outside the Core runtime dependency graph. The
gate is local/CI evidence; published SDK support still requires release
manifest, deprecation and external-consumer versioning decisions.
