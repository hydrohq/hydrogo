# HydroGo Contribution Rules

Before starting work that alters the architecture or product behavior in this repository, read the following documents:

1. `docs/HYDROGO_PHILOSOPHY.md`
2. `docs/implementation_plan.md`
3. `docs/architecture/README.md`
4. Affected ADR files

## Architectural Discipline

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
