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

The example is intentionally not a construction-management product. It is a
contract fixture used to verify that the Workspace can produce one safely.
