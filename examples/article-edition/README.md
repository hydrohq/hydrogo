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

## Local verification without a commercial license

`hydro serve` normally requires a vendor-signed `hydro.lic`. For a local
smoke check (for example `/healthz`) against a scaffolded project like this
fixture, build an ordinary `go build` binary (never an official vendor
release) and run:

```text
HYDRO_DEV_MODE=1 hydro serve --config config.yaml --address 127.0.0.1:PORT
```

(or pass `--dev` instead of the environment variable). This only bypasses
license verification when the binary has no vendor public key embedded and
the server binds to `127.0.0.1`/`localhost`/`::1`; official/customer
binaries always have the vendor key embedded, so `--dev`/`HYDRO_DEV_MODE`
has no effect on them. See `docs/architecture/0336-dev-mode-license-bypass.md`.
