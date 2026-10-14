# HydroGo API and Module Development Guide

This guide outlines verified boundaries for developing a Client Edition, module, or extension
without forking HydroGo Core. Application code must never couple directly to SQL, cache, or queue
drivers; these details remain encapsulated inside repository and adapter contracts.

## Client Edition and Extensions

Contracts under `extension/v1` represent candidate-stable APIs. New capabilities are added first to
a versioned contract; existing method signatures must never change silently. Breaking changes require
a new major version, a deprecation alias, and migration notes. Edition-specific behavior belongs in
`custom/` or within the relevant module boundary; HydroCore is never forked for customer needs.

## Registering Routes

Every route is registered with an explicit access classification:

```go
router.Route{
    Pattern: "GET /public/example/{slug}",
    Public:  true,
    Handler: http.HandlerFunc(handler),
}
```

Never conflate public, protected, and admin routes. When adding a new route, add test cases in the
same package covering the access class, unauthorized requests, invalid HTTP methods, and safe error
messages. Public projections must never return draft, deleted, or scheduled content.

## Handler and Service Boundaries

Handlers govern authentication, CSRF, request body limits, input validation, and HTTP status codes.
The domain/service layer manages business rules and transaction boundaries. The repository layer
abstracts SQL dialects. Error responses must never leak driver internals, raw SQL, stack traces, or
secrets; operational details are correlated via request IDs in structured log and audit contexts.

Write tests for optimistic locking, idempotency, and retry ambiguity in new write flows. External
provider calls must enforce bounded timeouts and exponential backoff, producing replayable outcomes.

## Migrations and Inactive Modules

Migrations must verify SQLite and PostgreSQL behaviors independently, including rollback and compatibility
notes. An inactive module must not register routes, execute migrations, serve assets, spawn workers, or
consume mandatory system resources. Include module enablement and disablement in `cmd/hydro` preset smoke tests.

## Verification Workflow

Before merging breaking changes, compare OpenAPI specifications:

```powershell
hydro validate api --baseline .\api-baseline.json --current .\generated\resource\openapi.json
```

The check catches removals of published paths, operations, responses, or properties fail-closed;
semantic compatibility of business logic remains subject to module code review.

```powershell
go test ./...
go test -race ./...
pwsh ./scripts/fuzz-smoke.ps1
pwsh ./scripts/verify-local.ps1
```

Run `git diff --check` and `gofmt` before committing code changes. Retain `hydro doctor`, SBOM,
and vulnerability scan outputs prior to releases.

## Troubleshooting

- If `/readyz` fails, inspect `hydro doctor` and migration status first.
- If uploads are rejected, verify size limits, MIME types, decoded pixel boundaries, and storage directory permissions; temporary files must not leak into permanent storage.
- Respect `Retry-After` on `429` responses; Redis rate limiter outages must fail closed rather than bypass limits.
- If generator output differs from expectations, verify the `hydro new --frontend-manifest` JSON contract, unique route names/paths, and `test_id` mappings.

Production PostgreSQL/Redis failovers, backup-restore rehearsals, blue/green rollbacks, high-concurrency capacity drills, and external penetration tests fall outside the scope of this local guide; corresponding open gates are maintained in `PROFESSIONALIZATION_BACKLOG.md`.
