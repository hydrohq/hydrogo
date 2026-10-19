# Contributing to HydroGo

Thanks for helping. Short version: open an issue for anything bigger than a bug fix, keep changes small, and run the checks.

- **Read first:** [AGENTS.md](AGENTS.md) holds the architecture rules (JSON-only Go, one migration per dialect, no customer-specific content in core, zero build on publish). It is written for AI agents but applies to everyone.
- **Set up:** Go 1.27+ and Node 22. `go run ./tools/devserver` starts the server with demo data at http://127.0.0.1:8090 (admin at `/admin`).
- **Branches:** `main` is the release branch: it only receives finished work and release tags (`vX.Y.Z`) are cut from it. Day-to-day work goes to `development`, which has no GitHub CI; run the checks locally. CI runs on pull requests into `main` and on pushes to `main`.
- **Before you push:** `go run ./tools/check` (or `make check`). It builds, vets, formats, runs the guards, Go tests and the admin lint/typecheck/tests/build. If you changed admin UI, open the screen and look at it (see AGENTS.md).
- **Tests:** add a test next to the code you change. Do not skip or rename failing tests; fix them or explain a deletion in the commit message.
- **Commits and PRs:** explain the why in the commit message. Describe what a user would notice: Before, After, then how. Link the issue.
- **New content types** are YAML only (`content-types/`); see [docs/content-types.md](docs/content-types.md). **New migrations** need all three dialects.
- **Security issues:** do not open a public issue; see [docs/SECURITY.md](docs/SECURITY.md).

By contributing you agree your work is licensed under the MIT license of this repository.
