# ADR-0149: Shared Multi-Tenant SaaS Evolution & Isolation Architecture

- **Status:** Accepted (Phase 6 Milestone)
- **Deciders:** HydroGo Engineering & Architecture
- **Date:** 2026-09-19
- **Depends On:** [ADR-0140](0140-tenant-isolation-scope.md), [ADR-0147](0147-single-tenant-production-deployment-and-bff-model.md)

## Context

Following successful single-tenant production deployments (Phases 0–5), expanding HydroGo into a shared-database multi-tenant SaaS requires a rigorous security and isolation model. Multi-tenancy in HydroGo is not a cosmetic database column; it is a fundamental authorization and data lifecycle boundary.

## Decision

1. **Explicit Tenant Context Propagation:**
   - Every request resolves a `TenantContext` at the edge router (`internal/router`).
   - The context is carried explicitly through:
     `HTTP Handler -> Application Service -> Repository -> Jobs / Outbox -> Cache Key -> Object Storage Path -> Audit Log`.
2. **Fail-Closed Repository Enforcement:**
   - Any database query executed without an explicit `tenant_id` in a multi-tenant build will immediately abort and panic/fail-closed (`ErrMissingTenantContext`).
   - PostgreSQL Row-Level Security (RLS) is used as defense-in-depth with `SET LOCAL app.current_tenant_id = '...'`.
3. **Storage & Cache Isolation:**
   - **Distributed Cache:** Every Redis/Valkey key is deterministically scoped: `t:{tenant_id}:{module}:{key}`.
   - **Object Storage:** Media assets are partitioned by tenant prefix: `s3://bucket/tenants/{tenant_id}/media/...`.
   - **Job Queues:** Background workers execute with tenant context bound to the job payload; cross-tenant job processing without bound context is forbidden.
4. **Encryption Key & Data Sovereignty:**
   - Column encryption keys (MFA secrets, API credentials) are derived using tenant-specific key-derivation functions (HKDF with per-tenant salt).
   - "Right-to-be-Forgotten" and tenant deprovisioning triggers a cryptographic erase of the tenant's primary key, rendering past encrypted backups unrecoverable.
5. **Cross-Tenant Administration:**
   - Super-admin operations across tenants require an explicit, audited cross-tenant scope (`RoleSuperAdmin` with dedicated audit logging).
   - Standard administrative roles are strictly confined to their own tenant context.

## Release Gate

Tenant isolation property tests, IDOR fuzzing, noisy-neighbor load tests, and database fail-closed verification must achieve 100% pass rate before enabling shared multi-tenant SaaS mode.
