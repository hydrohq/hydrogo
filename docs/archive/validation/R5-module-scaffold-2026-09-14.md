# R5 — Module authoring scaffold validation

**Status:** Local contract succeeded.

The module authoring command now emits model, repository, service, policy,
transport, dialect migration, contract/fixture tests, admin manifest, and
module metadata. Existing deterministic output, secret-field denylist,
symlink/path checks, and immutable migration protections remain in force.

Evidence: `go test ./internal/generator ./cmd/hydro` passed on 2026-09-14;
generator tests compile generated fixtures and assert the new scaffold files.
Full repository pre-push verification is required before delivery.

The canonical starter input is `examples/module-starter/collection.yaml` and
is loaded/validated by `TestModuleStarterFixtureLoadsAndValidates`.
