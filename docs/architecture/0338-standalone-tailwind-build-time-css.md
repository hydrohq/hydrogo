# ADR-0338: Standalone Tailwind CLI for isolated, build-time custom UI CSS

- **Status:** Accepted
- **Date:** 2026-10-02
- **Supersedes:** [ADR-0041](0041-admin-visual-theme-modes.md) (its "no Tailwind" consequence only; its four admin appearance modes are unchanged and remain governed by that ADR)
- **Superseded by:** None
- **Related decisions:** [ADR-0041](0041-admin-visual-theme-modes.md), [ADR-0053](0053-semantic-ui-token-contract.md), [ADR-0020](0020-headless-first-admin-studio-and-frontend-templates.md)

## Context

ADR-0041 ruled out Tailwind for the Admin Studio to avoid adding a runtime
frontend dependency to a server-rendered panel. That reasoning still holds
for `internal/ui` and the four existing visual families (`hydrogo`, `shadcn`,
`workbench`, `nova`), which stay on the ADR-0053 semantic token contract.

Separately, teams building Client Edition-specific pages under `custom/`
(policy, workflow, and integration surfaces per AGENTS.md's generated/custom
split) want a faster way to produce one-off, non-shared layouts without
inventing ad-hoc CSS by hand each time, and AI-assisted authoring tools
produce materially better results with Tailwind's utility vocabulary than
with hand-rolled class names. This repo has no Node/npm toolchain today and
is a self-contained Go binary by design; any solution must not change that.

Tailwind v4 ships a standalone CLI (a single native executable, no Node
required) that compiles a CSS entry file plus `@source`-scanned template
files into static CSS at build time. This is a different risk profile than
the "runtime Tailwind/CDN" option ADR-0041 rejected.

## Decision

1. A new, isolated source tree, `custom/ui/tailwind/`, is the only directory
   the Tailwind build ever scans:
   - `custom/ui/tailwind/input.css` — the Tailwind entry file
     (`@import "tailwindcss";` plus `@source` directives scoped to
     `custom/ui/tailwind/example` and any future sibling directories added
     the same way).
   - `custom/ui/tailwind/example/` — Templ components authored with real
     Tailwind utility classes (`card.templ` / generated `card_templ.go`),
     serving as the working reference and as the subject of the regression
     test in Validation.
   - `custom/ui/tailwind/dist/tailwind.css` — the compiled, minified output.
   - `custom/ui/tailwind/embed.go` — a `//go:embed dist/tailwind.css`
     package (`package tailwind`) exposing `CSS() (string, error)`, mirroring
     the existing pattern in `static/embed.go`.
2. The standalone Tailwind CLI binary is a **build-time-only** tool, invoked
   by `scripts/build-tailwind.ps1` (mirrors `scripts/theme-token-lint.ps1`'s
   style). It is **not** vendored into the repo and **not** a dependency of
   `go build`: the compiled `dist/tailwind.css` is committed, the same way
   `static/vendor/htmx`, `.../inter`, and `.../quill` commit third-party
   output rather than requiring their build tooling at Go build time. A
   contributor only runs the script, and needs the CLI on their machine,
   when they change a file under `custom/ui/tailwind/`.
3. `internal/ui`, the four existing admin visual families, `AdminCSS`, and
   the ADR-0053 semantic token contract are **not modified**. The compiled
   Tailwind stylesheet is not linked from `internal/ui/admin_shell.go` or any
   existing route. It is only ever loaded by pages that explicitly import
   `custom/ui/tailwind` and its own components — today, nothing does; the
   `example` package exists solely to prove the pipeline and anchor the
   contract test. A real Edition page opts in by importing the package and
   rendering `<link>`/inline `<style>` with `tailwind.CSS()` itself.
4. Tailwind-authored pages must never load the compiled Tailwind stylesheet
   on the same page as `AdminCSS` or any of the four admin families. Tailwind
   v4 defines common-named CSS custom properties (e.g. `--radius-lg`,
   `--shadow-md`) that would collide with the ADR-0053 semantic token names
   if both stylesheets were active in the same cascade. This is a hard
   constraint on where the `custom/ui/tailwind` output may be loaded, not a
   theoretical concern.

## Alternatives

- **npm + Node.js Tailwind pipeline.** Rejected: introduces a second language
  toolchain to a pure-Go project, with its own `node_modules`, lockfile, and
  CI install step. The standalone CLI gets the same compiled-CSS outcome
  without it.
- **Vendor the standalone CLI binary under `static/vendor/`, like htmx.**
  Rejected: the standalone binary is roughly 100 MB per OS/architecture
  (Windows/macOS/Linux × amd64/arm64), versus kilobytes for htmx/Quill. That
  is a disproportionate repo-size cost for a dev-only build tool whose output
  is what actually ships. Documenting the download step in
  `scripts/build-tailwind.ps1` and this ADR is cheaper and keeps the CLI
  version a conscious, visible per-regeneration choice.
- **Retrofit Tailwind into the existing four admin visual families.**
  Rejected outright and out of scope: it would contradict ADR-0053, require
  rewriting `internal/ui`'s shared Templ components, and repeat the exact bug
  documented at `internal/ui/admin_shell.go:2747-2756` (Tailwind utility
  class names used with no matching CSS) at a much larger blast radius.
- **Generate CSS for the whole repo from one global Tailwind config.**
  Rejected: would require `@source` scanning across `internal/ui` and every
  theme, defeating the isolation this ADR relies on to avoid touching
  ADR-0053.

## Consequences

### Benefits

- Edition-specific, non-shared UI work gets a utility CSS vocabulary that
  AI-assisted authoring tools handle well, without touching Core's
  semantic-token UI contract.
- Zero runtime or JS footprint: the shipped artifact is static CSS text,
  embedded like any other static asset.
- `go build ./...` never requires the Tailwind CLI to be installed.

### Costs and risks

- A second, parallel styling vocabulary now exists in the repository
  (semantic tokens for Core/Admin, Tailwind utilities for `custom/ui/tailwind`
  pages). This is intentional and bounded by directory, but must stay that
  way — any PR that imports `custom/ui/tailwind` into `internal/ui` or an
  existing admin route should be rejected in review as an ADR-0053/ADR-0041
  violation.
- The compiled `dist/tailwind.css` can silently drift from
  `example/*.templ` if a contributor edits classes without rerunning
  `scripts/build-tailwind.ps1`. Mitigated by
  `TestExampleComponentClassesAreCompiled` (see Validation), which fails the
  build when a used utility class has no matching compiled rule.
- Tailwind's reset layer and custom-property names are not namespaced;
  loading the compiled CSS alongside `AdminCSS` would corrupt admin theming.
  Mitigated by keeping the two stylesheets structurally unable to load
  together (no shared template, no shared route) and documenting the
  constraint here.

## Scale impact

- Compact: no runtime impact — the asset is static text embedded in the
  binary, served like any other static file; no new worker, route, or
  service is added by this ADR alone (a future Edition page that uses it
  would add its own route, same as any other page).
- Standard: no change.
- Distributed: no change; the embedded asset is part of the binary and needs
  no shared state or cache coordination across instances.

## Database impact

- SQLite: N/A — no schema, query, or storage change.
- PostgreSQL: N/A
- Future adapters: N/A

## Cache impact

N/A — no server-side cache key, TTL, or invalidation is introduced. The
compiled CSS is a static embedded asset; HTTP-level caching for it (if and
when a real route serves it) follows whatever static-asset caching the
serving route already uses, same as other `static/vendor` assets.

## Deployment and compatibility

Purely additive: a new isolated directory, a new build script, and a new Go
package with no imports from and no imports into `internal/ui`. No migration,
no route change, no behavior change to any existing page. Old and new
binaries can coexist during a Blue/Green rollout without any compatibility
shim, since nothing existing references `custom/ui/tailwind`.

## Validation

- `custom/ui/tailwind/embed_test.go`:
  - `TestEmbeddedTailwindCSSIsPresent` — the embedded asset is non-empty.
  - `TestExampleComponentClassesAreCompiled` — every utility class used by
    `custom/ui/tailwind/example/card.templ` has a matching compiled selector,
    guarding against the dead-class bug class fixed at
    `internal/ui/admin_shell.go:2747-2756`.
  - `TestExistingAdminVisualFamiliesAreUntouched` — `internal/ui.AdminCSS`
    (and, by extension, the hydrogo/shadcn/workbench/nova families it backs)
    remains present and unmodified.
- `go build ./...` and `go test ./...` pass with no changes to any existing
  `internal/ui` test.
- `scripts/theme-token-lint.ps1` only scans `themes/` and is unaffected by
  this change (confirmed: still passes, same definition/reference counts as
  before this ADR).
- Regeneration command: `pwsh scripts/build-tailwind.ps1 -TailwindCli <path-to-standalone-binary>`
  (binary from https://github.com/tailwindlabs/tailwindcss/releases, asset
  `tailwindcss-<os>-<arch>[.exe]`).

## References

- ADR-0041, ADR-0053
- `docs/UI_DESIGN_SYSTEM.md`
- `static/embed.go` (embed pattern this ADR mirrors)
- `scripts/theme-token-lint.ps1` (script style this ADR mirrors)
