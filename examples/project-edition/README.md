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

The `milestone.blueprint.yaml` file is a second resource produced with
`hydro new resource`. Its metadata enters the shared Admin registry
automatically through the already-wired `generated.RegisterGeneratedAdmin`
helper. The Edition still explicitly composes its API route with an
Edition-owned policy and applies the reviewed SQLite/PostgreSQL migration in
the migration command; Blueprint generation never guesses those security or
database activation decisions. The Chromium and SQLite/PostgreSQL CRUD evidence
is recorded in `docs/validation/blueprint-derived-admin-activation-2026-10-07.md`.

The example is intentionally not a construction-management product. It is a
contract fixture used to verify that the Workspace can produce one safely.
Its server uses permissive demo policy callbacks and is restricted to loopback
addresses. Replace those callbacks with Edition authentication, authorization,
CSRF, and tenant rules before adapting the composition to a deployable Edition.
