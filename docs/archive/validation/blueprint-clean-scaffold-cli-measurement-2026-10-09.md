# Clean scaffold Blueprint preparation measurement — 2026-10-09

## Scope

This measures the warm local CLI preparation path for a compact Client Edition:
project scaffold, one `article` Blueprint resource, and dependency-free project
check. A built CLI binary is reused across each three-step run. It does not
measure source compilation, database migration approval/application, generated
Edition server composition, authentication or policy wiring, or first CRUD.

## Results

| Run | `project new` | `resource create` | `project check` | Output after resource generation |
| --- | ---: | ---: | ---: | --- |
| 1 | 16.5 ms | 18.3 ms | 15.2 ms | 70 files, 15 Go files, 74,417 bytes |
| 2 | 15.1 ms | 18.6 ms | 15.8 ms | 70 files, 15 Go files, 74,417 bytes |
| 3 | 15.1 ms | 17.6 ms | 15.6 ms | 70 files, 15 Go files, 74,417 bytes |
| Median | **15.1 ms** | **18.3 ms** | **15.6 ms** | deterministic counts |

Commands per run:

```sh
hydro project new edition --root <empty-parent> --preset=compact-site \
  --profile=compact --no-demo
cd <empty-parent>/edition
hydro resource create --name article --field title:text --out .
hydro project check --config config.yaml
```

The generated resource defaults include `title` as its document title and
visible list/search field, plus the API/Admin helper and a
`generated/article/RUNTIME_WIRING.md` guide. Repeated outputs had identical
counts and project check passed in all three runs.

## Native first-CRUD gap

The scaffold command reported `core=licensed-runtime`. Inspection found no
project `go.mod` and no `cmd/server/main.go`; the Go files were Blueprint-
generated resource/helper artifacts, not a runnable Edition. The customer must
compose those artifacts into an authenticated runtime, bind tenant/resource
policy and CSRF, review and explicitly apply migration proposals, then run a
browser CRUD journey. The current CLI measurement does not time or pass those
steps and cannot be compared to the existing `--with-collections` browser
bridge.

ADR-0091 remains Proposed and defines private runtime artifact distribution
through a signed artifact/license broker. This measurement leaves that
architecture decision unresolved; no runtime download, source exposure,
authorization default, or automatic DDL was introduced. A clean scaffold to
native CRUD result remains an open Phase 3/5 acceptance gate.
