# HydroGo Contribution Rules

Before starting work that alters the architecture or product behavior in this repository, read the following documents:

1. `docs/HYDROGO_PHILOSOPHY.md`
2. `docs/INFORMATION_HIERARCHY.md`
3. `docs/implementation_plan.md`
4. `docs/architecture/README.md`
5. Affected ADR files

## Architectural Discipline & Respect for the User

- Never expose unnecessary technical noise, internal errors, cryptographic algorithms, or cognitive clutter to end users/members. Follow `docs/INFORMATION_HIERARCHY.md` strictly across all user, admin, sysadmin, developer, and auditor surfaces.
- Do not silently apply changes that contradict an accepted ADR.
- If new information supersedes an old decision, do not delete the old ADR; create a new ADR, mark the previous decision as "Superseded", and link the two documents.
- Never fork HydroCore for a specific customer need. Place general capabilities in versioned Core/Module contracts and specific customer behaviors in Client Editions.
- An inactive module must not create routes, migrations, assets, workers, or mandatory resource overhead.
- Never leak SQL dialect or cache driver details into domain and application service code.

## Change Verification

For a new or modified module, evaluate the following:

- SQLite/PostgreSQL contract behavior and impact on future adapters
- Dialect migrations and rollback/compatibility metadata
- Cache key, TTL, invalidation, stampede protection, and failure fallback
- Preventing unnecessary services and unbounded resource consumption in the Compact profile
- Connection pool, worker, and backpressure behavior in larger profiles
- Old and new version compatibility during Blue/Green deployments
- Continuity of sessions, media, assets, and background jobs
- Security, auditability, and data lifecycle management

## Definition of Done

An architectural change is not considered complete until its corresponding ADR, implementation, certified database behavior tests, required migration metadata, and capacity/deployment impact documentation are all delivered together.

## Workspace Generation and AI UI Rules

HydroGo is a Workspace/Engine that produces Client Editions; it is not a
customer-specific CMS, ERP, or construction product. Core changes must remain
generic and versioned. Customer behavior belongs in an Edition or module.

- Keep machine-owned output under `generated/`; keep human-owned policy,
  workflow, authorization, and integration code under `custom/`. Never edit a
  generated file to implement business logic; change the blueprint or
  generator instead.
- Server-rendered UI must use typed Design System components. Prefer Templ
  components such as `ui.Button`, `ui.Input`, `ui.Card`, and `ui.Form` over
  ad-hoc raw HTML. Components own escaping, accessibility defaults, and visual
  tokens.
- HTMX is an explicitly allowlisted enhancement layer, not a client-side state
  replacement. Use only registered attributes and contracts; every mutation
  must declare target, swap, CSRF, authorization, tenant scope, and idempotency
  behavior. New attributes require a component contract or ADR.
- Never use React-style `className` or stringly concatenated markup in Templ.
  Use `class` and typed class/token helpers. Do not introduce a frontend
  runtime dependency into Core or the Compact profile for a generated view.
- Blueprints may generate migration proposals, typed models, repository
  interfaces, route metadata, SDK contracts, and test skeletons. They must
  never execute DDL, invent authorization policy, or silently add a worker,
  route, asset, or service to an inactive module.
- Development conveniences such as `templ --watch`, `air`, and live reload
  belong only to `hydro dev`; production binaries must remain self-contained
  and dependency-light.
- All generated migrations require SQLite/PostgreSQL compatibility metadata,
  rollback/recovery notes, and certified contract tests before an Edition can
  enable them.
