# 0011: Generate per-role permission tests from content types

- Status: Accepted (Phase 1 and 2); Proposed (Phase 3)
- Date: 2026-10-23

## Context

HydroGo's whole pitch is that a content type is declared once in YAML and a
project gets storage, an admin API and an admin screen for free, with no
Go/Svelte code. Nothing, however, generates tests from that same
declaration — a project author (or the AI agent writing the YAML for them)
has had to hand-write a test, every time, to prove a new content type's
declared permissions (`Permissions.Read`/`Write`, per `internal/contenttypes`)
actually hold for every admin role. That's exactly the kind of boilerplate
the declarative model is supposed to remove.

Two things the code already does shaped this:

1. **A `hydro new` project has no Go module** (`internal/scaffold.Create()`
   writes `config.yaml` + `content-types/*.yaml` + optionally a JS/TS
   `frontend/`, never a `go.mod`). So "generated Go tests in the project"
   isn't coherent — generated tests have to be TypeScript/Playwright,
   black-box, against a running `hydro serve`. This also means they test the
   actually-enforced boundary (a real login, a real HTTP call, a real status
   code), not a mocked permission check.
2. **`internal/contenttypes.AdminAPI.resolve`** calls its permission check
   (`allowed`) strictly before decoding the request body or touching
   business validity (`internal/contenttypes/admin_http.go`'s `write`
   function). So a permission test needs no seeded business data — a
   generated test can always tell "was this request authorized" apart from
   "was this payload valid," which is what keeps it safe to regenerate on
   every schema change without knowing the content type's business rules.

## Decision

`hydro generate tests` (new, `internal/testgen` + `cmd/hydro/generate_tests_cmd.go`)
reads a project's `content-types/*.yaml` the same way the server does
(`contenttypes.LoadDir`) and writes one Playwright spec per content type
into `e2e/generated/content/`, proving every built-in role (admin, editor,
viewer) gets exactly the read/write access its declared permissions say it
should. Scope is deliberately the access-control matrix, not full business-flow
QA: verifying a create actually saves *valid* data needs domain knowledge the
schema alone doesn't carry (a real company id, a real deal amount); those
journeys stay hand-written in `e2e/custom/`, never touched by the generator.

`hydro new` now always scaffolds an `e2e/` harness (`internal/scaffold`), and
`hydro seed-test-users` (new) creates one fixed account per built-in role,
idempotently, printing the `HYDRO_E2E_<ROLE>_EMAIL`/`_PASSWORD` the generated
specs read.

## A real rate-limit collision, found by actually running it

The first version logged in fresh in every single test. Run with Playwright's
default `fullyParallel: true`, that instantly tripped HydroGo's own login
rate limit (`newRateLimiter(5, time.Minute)` in `cmd/hydro/serve_runtime.go`,
5 attempts/minute per IP, shared across every role since all requests came
from the same test runner) — every test but the first few failed with 429,
not a permission result. Fixed by logging in **once per role for the whole
run**: `e2e/global-setup.ts` (scaffolded alongside `playwright.config.ts`)
logs in once per seeded role and saves the session via Playwright's
`storageState`; `_helpers.ts`'s generated `login()` reuses it instead of
calling `/auth/login` again. Verified against a real `hydro serve` instance:
24 generated tests across 4 real CRM content types, 8 parallel workers, 3
total logins, all passing in under a second.

## Verified against a real server, not just unit tests

Beyond the unit tests in `internal/testgen` and `cmd/hydro`, this was run
end-to-end against a scaffolded `hydro new --preset crm` project with a live
`hydro serve`: seeded the three role accounts, generated the specs, ran them
for real (24/24 passing). Then a mutation check — added a custom
`permissions: {read: crm.deals.read, write: crm.deals.write}` to `deals.yaml`,
regenerated, and reran: the live server really does return 403 for editor and
viewer exactly as the regenerated test now expects, and admin still passes.
A second check confirms the tests aren't vacuously green: deliberately
flipping one expected status (403 → 200) makes that test fail with the real
observed value in the diagnostic, not silently pass.

## Phase 3 (proposed, not yet built)

Browser-driven admin UI specs (per role: log in, see/not-see the content
type in the nav, see/not-see create-edit-delete controls) walking the same
`Definition.AdminCollection()` metadata the admin SPA itself renders from.
Deferred: it needs the admin's generic form field selectors to be a stable,
documented contract first (today `GenericFormView.svelte` renders
`[name="<field>"]` inputs, which `tools/e2egen` already assumes for its own,
narrower manifest-driven tests — worth confirming that contract explicitly
before a second generator depends on it).

## Consequences

- A new content type gets a real, passing-or-failing permission test with
  zero hand-written code the moment `hydro generate tests` runs — matches
  the "declare it once, get it for free" model the rest of HydroGo follows.
- New public CLI surface: `hydro seed-test-users`, `hydro generate tests`.
  Both no-ops on a project with no content types or no `--config`-resolvable
  database; neither touches `e2e/custom/`.
- Risk: the generator's assumption that permission checks run before payload
  validation is specific to `internal/contenttypes.AdminAPI.resolve`'s
  current order of operations. If that order ever changes, the generated
  write-tests' `not.toBe(403)` assertions for allowed roles would need
  revisiting — flagged here so a future change to `resolve`/`write` checks
  this decision record.
