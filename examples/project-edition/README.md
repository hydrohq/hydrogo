# Project Client Edition example

This is a small Client Edition fixture showing HydroGo's central architectural
promise: a project-specific product is assembled at the Edition boundary while
the Core remains reusable. `generated/` contains machine-owned schema,
repository, API and view artifacts; `custom/` contains the policy/workflow
extension point where construction-project rules would belong.

This example demonstrates the Workspace flow without modifying HydroGo Core:

```text
hydro generate scaffold --blueprint blueprint.yaml --out .
hydro project check
```

The command produces `generated/project` and preserves this `custom/` policy
boundary for tenant ownership, workflow, and authorization decisions.

The `milestone.blueprint.yaml` and `article.blueprint.yaml` files are additional
resources produced with the Blueprint scaffold. Article's rich-text field uses
the generated `text` column and the shared Quill editor. Its embedded vendor
assets are mounted by the standalone example server under `/vendor/`, matching
the Core server's asset route. Its metadata enters the shared Admin registry
automatically through the already-wired `generated.RegisterGeneratedAdmin`
helper. API routes are composed once through the generated
`RegisterGeneratedResources` helper and an Edition-owned policy callback, so
new generated resources join the API and Admin registry without per-resource
route imports. Tenant resolver callbacks remain required for tenant-scoped
resources. The Edition still applies each reviewed SQLite/PostgreSQL migration
in the migration command; Blueprint generation never guesses security or
database activation decisions. The Chromium and SQLite/PostgreSQL CRUD evidence
is recorded in `docs/validation/blueprint-derived-admin-activation-2026-10-07.md`.

`content-block.blueprint.yaml` adds a separate `content_block` resource with
an Editor.js body and live, CSRF-protected preview. Its Edition-owned bundle,
sanitization boundary, paragraph-only support, and SQLite/PostgreSQL migration
are documented in ADR-0363 and
`docs/validation/generated-edition-block-editor-2026-10-08.md`.

The example is intentionally not a construction-management product. It is a
contract fixture used to verify that the Workspace can produce one safely.
Its server uses permissive demo policy callbacks and is restricted to loopback
addresses. Replace those callbacks with Edition authentication, authorization,
CSRF, and tenant rules before adapting the composition to a deployable Edition.
