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

## Local verification

To run a local smoke check (for example `/healthz`) against a scaffolded project like this fixture:

```text
hydro serve --config config.yaml --address 127.0.0.1:PORT --dev
```

(or run with `HYDRO_DEV_MODE=1`). See `docs/architecture/0336-dev-mode-license-bypass.md`.
