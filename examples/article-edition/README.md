# Article Client Edition example

This fixture demonstrates how a content/article product is produced from the
same Blueprint pipeline as a project Edition. It does not fork HydroCore and it
does not execute DDL while the application is running.

Generate or refresh the reviewable collection artifacts with:

```text
hydro generate scaffold --blueprint blueprint.yaml --out .
hydro resource check --root .
```

Generated artifacts are deterministic and reviewable; editorial workflow,
tenant policy and integrations remain human-owned under `custom/`.
