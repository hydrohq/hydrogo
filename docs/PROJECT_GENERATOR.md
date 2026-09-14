# HydroGo Project Generator

`hydro new` creates a client-owned project scaffold without copying HydroGo Core source code.

## Usage

```powershell
hydro new haber-merkezi --preset=news
hydro new magaza --preset=commerce --database=postgres --profile=standard
hydro new portal --preset=portal --git
hydro new haber-merkezi --preset=news --frontend-manifest .\dist\frontend-manifest.json
```

The command creates configuration, `hydrogo.lock`, a runnable minimal theme, project migration and custom-extension placeholders, seed guidance, and a metadata-driven E2E manifest. `--git` runs `git init` only inside the newly created directory; it never adds a remote or pushes.

Supported presets are `news`, `editorial`, `commerce`, `compact-site`, `corporate`, `portal`, and `headless`. `editorial` is the content-focused news profile; `compact-site` is the minimal pages/media/site profile. These presets keep unrelated modules disabled, select enabled feature contracts, and do not copy Core source code. The licensed HydroGo runtime is supplied separately and is referenced by the generated configuration and lock file.

## Module authoring

For a deterministic collection/module review surface, use the task-oriented
command:

```powershell
hydro module create --spec .\collection.yaml --out .\generated
```

The established spelling `hydro generate module --spec ... --out ...` remains
supported and produces the same validated artifacts. The collection directory
contains model, repository, service, deny-by-default policy, transport,
contract/fixture tests, a dialect migration draft, an admin manifest, and a
machine-readable `module.yaml`. Generated code is a reviewable starting point;
authorization, business rules, and migration rollout still require explicit
developer review.

### Frontend build manifest

When a frontend build produces a route contract, pass its JSON file with
`--frontend-manifest`. The file must contain `routes`, where every route has a
unique `name`, absolute safe `path`, and unique `test_id`; `public` is optional
and defaults to `true`. The generator strips unknown fields, sorts routes by
name, and writes the normalized contract to `e2e/manifest.json`. Invalid,
duplicate, or unsafe routes fail generation before a project is created.

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
