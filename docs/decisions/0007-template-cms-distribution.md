# 0007: Distribute HydroGo as a template CMS

- Status: Accepted
- Date: 2026-10-19

## Context

HydroGo is meant to be used by other people for their own sites, not only run as one deployment. Starting a project required Go and Node toolchains, the repository root carried one deployment's files, the default content types were news-portal specific, and there was no release, installer or upgrade story.

## Decision

1. **Releases.** Tagged `v*` releases publish prebuilt binaries (Linux, macOS, Windows; amd64 and arm64), a Windows `Setup.exe` that puts `hydro` on PATH, `install.sh`/`install.ps1`, `SHA256SUMS`, and a multi-arch image on GHCR. The admin panel stays committed in `internal/adminui/dist`, so release builds need only Go.
2. **Project scaffolding in the binary.** `hydro new DIR` / `hydro init` write a config and the content types of a preset (`internal/scaffold`, stock types embedded from `content-types/`) and can create the database and admin. Presets only choose modules and stock content types; they add no code paths.
3. **Neutral core.** Deployment-specific files and tools (a deployment script, a news-agency ingest bot, security scan reports, `PRODUCT.md`/`DESIGN.md`, `config.yaml`, `module-specs/` whose generator command no longer exists, `docs/archive`) are removed; they stay in git history.
4. **Frontend story.** `@hydrogo/client` (`packages/client`) is the typed API client; `examples/sveltekit-starter` is a server-rendered starter. `examples/svelte-starter` is HydroGo's own landing/docs SPA, and `examples/hydropost-news-starter` remains a reference for one news portal.
5. **Extension paths.** Supported ways to extend HydroGo: YAML content types, API keys and webhooks for integrations, and a frontend on the API. New Go modules are core changes, made in `modules/<name>` following `modules/news`.
6. **Versioning.** Semantic Versioning with changelog entries for every breaking change (see `docs/upgrading.md`).

## Consequences

Starting a project no longer needs the source tree. A change to a stock content type changes what new projects get; it never touches existing projects, whose copies live in their own `content-types/`. Release builds trust the committed admin build, so `make check` (which verifies it) must pass before tagging.
