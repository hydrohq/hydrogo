# Blueprint → Admin Initial CRUD Measurement

**Date:** 2026-10-07  
**Status:** Partial; total elapsed time from clean project scaffold to initial CRUD could not be measured.  
**Scope:** `hydro project new`, `hydro new resource`, `hydro project check`, and
actual generated Admin CRUD in a previously composed Project Edition.

## Clean Client Edition Scaffold Measurement

The HydroGo CLI was compiled once in a Linux environment; compilation time was not
included in measurements. Each repetition was executed in a distinct, empty directory using the
`editorial` preset. The resource command generated the `title:text:required,body:richtext`
Blueprint. Configuration and the generated registry were then verified via
`hydro project check --config config.yaml`.

| Run | `project new` | `new resource` | `project check` | Generated Resource Files |
| --- | ---: | ---: | ---: | ---: |
| 1 | 25 ms | 34 ms | 28 ms | 40 |
| 2 | 24 ms | 32 ms | 25 ms | 40 |
| 3 | 24 ms | 34 ms | 27 ms | 40 |
| Median | **24 ms** | **34 ms** | **27 ms** | — |

Commands were executed against a warm CLI binary; build/setup, external service provisioning,
migration execution, Go Edition composition, and browser CRUD time are excluded.
This measurement is not a Payload comparison and is not under identical conditions
to older scaffold measurements on Windows.

## Authentic Edition Activation Trial

A third resource (`milestone2`) was created in a temporary Project Edition worktree.
The generated Admin registry and `RegisterGeneratedAdmin` resource were automatically
updated from the Blueprint; no resource ID was added to the Admin menu.
The Edition's existing `demoPolicy` was only used in the loopback test fixture.
This policy must not be used in a production Edition.

Manual activation consisted of four edits across two Go files:

- `custom/composition.go`: importing the generated route package and calling
  `RegisterMilestone2Route`.
- `cmd/server/main.go`: importing the generated migration package and appending reviewed
  `MigrationSQL` to the explicit migrate command.

`go test ./examples/project-edition/... -count=1` passed. A temporary adapted copy of the existing
CRUD E2E scenario verified record creation, API readback, editing, and deletion in Chromium
via generated Admin: **passed 1/1** (13.3 seconds). This E2E copy was for experimentation; no new
test file was committed to the repository.

## Open Acceptance Item

`hydro project new` generates config/theme/migration directories, but does not produce a runnable
Go Edition module or Admin server composition. Therefore, on this clean scaffold, authentic browser CRUD
cannot be launched solely following `hydro new resource`. Valid authorization/tenant policy and migration
approval remain the responsibility of the Edition. Wall-clock time and active developer intervention time
from initial clean scaffold to working customer CRUD have not yet been measured.
The goal for subsequent development is to deliver a reproducible, tested starter/golden path that provides
Edition composition while maintaining these security boundaries.
