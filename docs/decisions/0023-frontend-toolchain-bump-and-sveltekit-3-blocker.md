# 0023: Admin toolchain bump; SvelteKit 3 blocked by an env-folding regression

- Status: Accepted
- Date: 2026-10-24

## Context

Dependabot opened nine PRs bumping Vite, TypeScript, `@sveltejs/kit`,
`@sveltejs/vite-plugin-svelte` and vitest across `admin`, `examples/crm-starter`
and `examples/sveltekit-starter`. None were individually mergeable: Vite,
`vite-plugin-svelte`, `@sveltejs/kit` and `@sveltejs/adapter-node` all have
peer-dependency ranges on each other's major version, so bumping one alone
breaks `npm install`. Investigating this surfaced a real, pre-existing
security issue: `crm-starter` and `sveltekit-starter`'s pinned `@sveltejs/kit`
(2.70.3, the latest 2.x release) is affected by two low-severity advisories
(GHSA-pfhg-q3qv-9q6r, GHSA-27mw-82g8-phjj — URL redirection via enhanced form
submission, XSS via an untrusted cache-control header on a prerendered page),
fixed only in `@sveltejs/kit` 3.x. There is no later 2.x patch.

## Decision

**`admin` was bumped as a coordinated set** (`vite@^8.3.5`,
`@sveltejs/vite-plugin-svelte@^7.3.1`, `svelte@^5.57.2`,
`typescript@^6.0.3` — not 7.x, since `svelte-check@4.7.7`'s peer range is
only `^5.0.0 || ^6.0.0`) and verified clean: `npm install` resolves with
zero `npm audit` findings, `svelte-check`/`vitest run`/`vite build` all
pass. `admin` has no `@sveltejs/kit` dependency (it is a plain Vite+Svelte
SPA), so it was not exposed to the issue below.

**`crm-starter` and `sveltekit-starter` were NOT bumped to SvelteKit 3**,
despite getting most of the way there (migrating `svelte.config.js` into
`vite.config.ts`'s `sveltekit({...})` call, `Handle`/`HandleServerError`
moving to `@sveltejs/kit/hooks`, param matchers moving from one-file-per-
matcher to a single `src/params/index.ts` exporting a `params` map, and
`cookie`'s major bump that both projects' own `overrides: { cookie:
"^0.7.2" }` was blocking). The migration hit a confirmed framework/bundler
regression: SvelteKit 3.1.0 paired with Vite 8's new default Rolldown
bundler constant-folds `$env/dynamic/private`/`$env/dynamic/public`
property reads (`env.HYDRO_API_ORIGIN`, `env.PUBLIC_SITE_ORIGIN`, etc.) to
`undefined` at build time instead of leaving them as a runtime lookup.
Verified by inspecting the actual compiled server bundle: `env.HYDRO_API_ORIGIN
?? 'http://127.0.0.1:8080'` in source compiled down to the bare literal
`"http://127.0.0.1:8080"` -- the environment variable silently never takes
effect in the production build, no matter what it is set to. This is not
cosmetic: a deployed starter would always talk to the hardcoded fallback
origin regardless of configuration. Forcing this upgrade into a template
meant to last years would trade a known, narrow, low-severity CVE for a
newly-introduced, silent, environment-wide correctness bug -- the wrong
trade.

Both starters were reverted cleanly to their last-committed state (Vite 6,
SvelteKit 2.70.3, verified `check`/`build`/`test`/`smoke` all still pass)
and the four blocked Dependabot PRs (#347, #348, #349, #351) were left
open with a comment explaining the exact regression and what to check
before retrying. #346 and #350 (admin's TypeScript 7 and Vite 8 bumps)
were closed as superseded by the coordinated bump above.

## Consequences

- `admin`'s toolchain is current and `npm audit`-clean.
- `crm-starter` and `sveltekit-starter` remain on `@sveltejs/kit@2.70.3`,
  which `npm audit` reports as affected by two low-severity advisories.
  This is a known, tracked, deliberately accepted risk, not an oversight:
  both starters are reference starting points a project copies and builds
  its own frontend from (`AGENTS.md`), not something HydroGo itself serves
  to the public, and both advisories require a specific interaction
  pattern (enhanced form submission redirect; a prerendered page serving an
  attacker-controlled cache-control header) that is narrower than a
  generic "visit any page" exposure.
- Revisit the SvelteKit 3 migration once either SvelteKit or Vite/Rolldown
  ships a fix for the env-folding regression above -- check by building and
  grepping the compiled server bundle for the literal fallback value this
  record found, the same way this investigation did, before trusting a
  green `svelte-check`/`vite build` alone (neither one caught this; only
  reading the actual compiled output did).

## Update (2026-10-24, same day): the `#lib` path is blocked too

Re-investigated in an isolated copy of `crm-starter` (same package set:
`@sveltejs/kit@3.1.0`, `vite@8.3.5`, `@sveltejs/vite-plugin-svelte@7.3.1`,
`@sveltejs/adapter-node@6.0.0`), specifically to test whether adopting
SvelteKit 3's actual replacement for `$lib` -- the new `#lib` import alias
-- avoids the `$env/dynamic/private` folding bug above by not needing a
custom `alias: { $lib: 'src/lib' }` at all (the earlier 0026 investigation
found that custom alias was what triggered the `"paths" was overwritten`
interaction with `$env`).

Renaming every `$lib` import to `#lib` across the whole starter and
building with no custom alias **does** fix the `$env/dynamic/private`
folding: the env var reads survive in the compiled output this time.
But `vite build` then fails a different way -- `Rolldown failed to
resolve import "#lib/..."`, naming a different, effectively random file
on each run. 13 consecutive clean-cache build attempts (including forcing
`UV_THREADPOOL_SIZE=1` and running `svelte-kit sync` as a separate step
first) all failed, never on the same import twice. This is a race
condition in Rolldown's own module resolver when resolving the `#lib`
alias, not anything reachable from our config -- `#lib` is SvelteKit 3's
own mechanism, implemented inside `@sveltejs/kit`/Vite, not something a
project configures.

Net: both documented paths to SvelteKit 3 on this toolchain are blocked
by a real upstream bug, for different reasons. Before retrying, check
*both*: grep the compiled bundle for folded `$env` literals (if still
using a custom `$lib` alias), and run `vite build` several times in a row
checking for `Rolldown failed to resolve import "#lib/` style failures
(if using the `#lib` default). A single green build is not enough
evidence for either -- the `$env` fold is silent and the `#lib` resolver
race only shows up on *some* runs.
