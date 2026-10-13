# HydroGo Prototype Velocity Plan

Objective: Rapidly produce initial working modules with Next.js-like agility while strictly
preserving production architectural contracts.

## Principles

- The generator emits only battle-tested, repeatable patterns; business logic remains explicit code.
- Server-rendered architecture is preserved by default; interactive islands and typed API clients are used when needed.
- Every generated component arrives with accessibility, authorization, migrations, and test skeletons.
- Core is never polluted with customer-specific logic; presets and Client Edition boundaries govern customizations.

## Phases

### V0 — Single-Command Development Environment

- `hydro dev`: Orchestrates SQLite, migrations, and seed workflows. `--dependencies auto` (default)
  starts disposable Redis/MinIO via Compose if available and tears them down on exit; if Docker is absent,
  it falls back gracefully to Compact profile. `--dependencies required` fails fast for CI/local gates,
  while `none` turns dependencies off completely. `--redis-url` explicitly overrides auto-discovered URLs.
- File watching, rapid template/CSS reloading, clean error formatting, and port conflict handling.
- `.env.example`, `hydro doctor --fix`, disposable test fixtures, and parity across WSL/Windows.

### V1 — Schema-Driven Generation

- `hydro generate resource|crud|page|api|admin|form` commands (emit deterministic review artifacts from the same reviewed contract).
- Generates models, migrations, repositories, services, policies, forms, accessible tables/empty states, APIs, OpenAPI specs, TypeScript clients, fixtures, and contract tests from a single schema.
- Emits explicit TODO boundaries, reference authorization checks, and review checklists.
- `hydro validate generator --spec SPEC --root OUTPUT` verifies lockfile hashes and expected generated artifact sets without writing to disk; suitable as an upgrade/pre-deploy CI check.

### V2 — Productized UI System

- Interoperability between shadcn-compatible light/dark themes and the existing theme engine.
- Layout presets for forms, data tables, filters, modals, drawers, toasts, wizards, uploads, empty/error states, and responsive viewports.
- Component showcase featuring RTL, long text strings, keyboard navigation, reduced-motion preferences, and high-contrast modes.
- Visual regression baselines and design token linting.

### V3 — Turnkey Product Presets

- Newsroom/publishing, e-commerce, corporate showcase, membership portal, dashboard, and ERP starters.
- Each preset includes navigation, authorization rules, seed data, demo screens, API endpoints, and deployment notes.
- Presets pass generator smoke tests and automated upgrade drills.

### V4 — Interaction and Data Layer Acceleration

- HTMX/HTML action feedback, optimistic UI updates, and progress indicators for long-running jobs.
- Typed API clients, standardized pagination/filter/sort parameters, caching, and invalidation template patterns.
- Bounded shared primitives for full-text search, data import/export, and bulk operations.
- Generator emits bounded CSV export/import, upload validation, and bulk/progress primitives; actual signed URL and storage policy remains at the module boundary.

### V5 — Team and Quality at Scale

- Module contract test generator, API compatibility checks, and selective mutation testing.
- ADR/template/documentation generators with CI-enforced drift detection.
- Telemetry on `time-to-first-route`, `time-to-first-crud`, build durations, and defect escape rates.
- SDK v1, semantic versioning, deprecation lifecycle, upgrade assistants, and backward-compatibility test suites.

## Acceptance Criteria

- From a clean project, scaffold the first public page and admin CRUD in under 15 minutes.
- Generated CRUD passes SQLite and PostgreSQL contract tests.
- Unauthorized access, CSRF, validation, migration rollback, and accessibility tests are automatically scaffolded.
- `hydro dev` provisions and orchestrates local dependencies with a single command.

`scripts/measure-iteration.ps1` measures UI contract compilation, full build, initial project scaffolding,
initial resource generation, and generated compatibility preflight times in a unified JSON report.
The report enforces the 15-minute acceptance threshold and runs as a clean-machine step in Windows CI builds.

Business-specific accounting/ERP logic, payment gateway selections, customer UX decisions, and production
credentials are never invented by the generator; they remain in Client Editions or separate ADRs.
