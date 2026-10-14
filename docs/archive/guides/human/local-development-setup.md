# Local Development Setup (Human Guide)

This guide covers running the HydroGo codebase locally for development and testing.

## Starting the Development Server

To launch the integrated development server with live reload:

```sh
go run ./cmd/hydro run
```

This single command orchestrates the development runtime:

- **Binds to `http://127.0.0.1:8090`** via a resilient development proxy. Navigate here in your browser.
- **Embedded SQLite:** Uses a local SQLite database (`data/demo.sqlite`) by default — zero external database installation needed.
- **Automatic Migrations:** Applies all pending database migrations automatically on boot.
- **Templ Auto-Watch:** Spawns a background watcher that re-renders `.templ` component changes instantly upon saving.
- **Live Binary Rebuild:** Automatically recompiles the Go application when Go source files change, presenting a smooth "Updating Server" interstitial screen rather than broken HTTP connections.
- **Demo Content Seeding:** Seeds sample articles, media, and a demo member account. *(Note: An admin account is not seeded automatically for security reasons. To create one, see [`docs/guides/human/admin-panel-access.md`](admin-panel-access.md).)*

## Making Changes & Iterating

1. Edit your source file and save.
2. The `hydro run` watcher detects the modification and triggers a build. Observe terminal output for compiler feedback.
3. Refresh your browser to inspect the changes.

## Codebase Orientation

- `cmd/hydro/` — CLI application entry point (`serve`, `run`, `migrate`, `create-admin`, etc.).
- `internal/` — Core engine packages, services, security boundaries, and transport handlers.
- `modules/` — Pluggable domain modules (`pages`, `news`, `members`, `forms`, etc.).
- `themes/` — Public site themes (layouts, blocks, and stylesheets).
- `migrations/` — Database migrations organized with dialect parity (SQLite, PostgreSQL, MySQL).
- `docs/architecture/` — Architecture Decision Records (ADRs).

## Key Development Rules

- **Never edit generated files directly:** Do not manually edit files ending in `_templ.go` or generated artifacts under `generated/`. Always modify the source `.templ` or blueprint definition.
- **Targeted testing during iteration:** During active development, run tests for the package you are working on (e.g. `go test ./modules/news/...`) rather than the entire multi-database suite. Run the full suite (`go test ./...`) before committing.
