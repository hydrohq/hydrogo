# ADR-0140: Multi-tenant scope is opt-in and explicit

- **Status:** Accepted (scope boundary); first Site Edition remains per-customer isolated deployment

## Decision

- A tenant is an authorization and data-lifecycle boundary, not merely a cache prefix.
- No multi-tenant routes, migrations, workers, or tenant context are activated in a single-tenant build.
- A future shared-database adapter must carry tenant scope through application services, repositories, jobs, media, cache keys, audit records, and backups; missing scope fails closed.
- Cross-tenant administration is a separate audited capability and is never inferred from a normal admin role.

## Release gate

Tenant isolation property tests, IDOR/role tests, restore/export tests, cache-key tests, noisy-neighbor capacity tests, and a PostgreSQL certification are mandatory before any SaaS/multi-tenant claim.
