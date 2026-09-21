# Project Client Edition example

This example demonstrates the Workspace flow without modifying HydroGo Core:

```text
hydro generate scaffold --blueprint blueprint.yaml --out .
hydro project check
```

The command produces `generated/project` and preserves this `custom/` policy
boundary for tenant ownership, workflow, and authorization decisions.

