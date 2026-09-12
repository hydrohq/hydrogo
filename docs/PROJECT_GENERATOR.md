# HydroGo Project Generator

`hydro new` creates a client-owned project scaffold without copying HydroGo Core source code.

## Usage

```powershell
hydro new haber-merkezi --preset=news
hydro new magaza --preset=commerce --database=postgres --profile=standard
hydro new portal --preset=portal --git
```

The command creates configuration, `hydrogo.lock`, a runnable minimal theme, project migration and custom-extension placeholders, seed guidance, and a metadata-driven E2E manifest. `--git` runs `git init` only inside the newly created directory; it never adds a remote or pushes.

Supported presets are `news`, `commerce`, `corporate`, `portal`, and `headless`. The preset selects enabled feature contracts and does not copy Core source code. The licensed HydroGo runtime is supplied separately and is referenced by the generated configuration and lock file.

## Workflow

1. Generate a project with `hydro new`.
2. Review `config.yaml` and `hydrogo.lock`.
3. Obtain a matching licensed HydroGo runtime and place `hydro.lic` beside the config.
4. Run `hydro migrate --config config.yaml` and `hydro create-admin --config config.yaml`.
5. Customize `themes/<name>`, add project behavior under `custom/`, and add additive project migrations.
6. Run the generated E2E suite and local verification before deployment.
7. Create a separate Git repository in the generated directory and connect its remote independently.

The current generator is a safe scaffold, not a source-code obfuscator and not a hosted deployment service. The binary executing the project still contains the runtime Core, so contractual licensing and binary distribution controls remain required.

## Generated and custom boundaries

Existing project directories are rejected; the generator never overwrites user files. Core migrations and runtime internals are delivered by the licensed runtime. Project-specific files belong in `custom/`, `themes/`, `seed/`, and project migrations.

## Licensing

Generated configurations mark licensing as required and bind `project_id`, edition, and features. Runtime verification uses the vendor trusted Ed25519 public key embedded in the official binary. A public key advertised inside a license file is never trusted. The private issuing key must remain outside customer projects and customer servers.

The commercial license text is a legal draft and requires lawyer review before customer distribution.
