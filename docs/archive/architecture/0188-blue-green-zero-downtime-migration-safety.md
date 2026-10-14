# 0188. Blue/Green Zero-Downtime Deployment & Migration Safety Certification

Date: 2026-09-20
Status: Accepted

## Context

HydroGo is engineered for enterprise high availability, continuous zero-downtime deployment, and safe multi-tenant operation. When deploying new releases to production (whether under single-tenant dedicated deployments or shared multi-tenant SaaS environments), older running binaries and newly deployed binaries frequently run concurrently during blue/green traffic cutover and canary rollouts.

To guarantee zero downtime and prevent operational data corruption or request failures during these transition windows:
1. All database migrations must follow strict expand/contract lifecycle discipline.
2. In the "expand" phase, new columns, tables, and indices must be backward-compatible with previous binary versions (e.g., using nullable columns or default values, non-destructive index creation).
3. Old binaries must continue to read and write without crashing or corrupting data written by new binaries.
4. Concurrent migration runners must serialize safely using transactional lock tables/advisory locks without deadlocking application traffic.
5. All migration files across modules and dialects (SQLite, PostgreSQL) must satisfy automated CI safety gates (`migrationguard`) validating sha256 checksums, dialect compatibility, disk usage estimates, lock timeout constraints, and reversible/expand rollback semantics.

## Decision

1. **Certified Zero-Downtime Schema Evolution:**
   - Enforce bidirectional compatibility across consecutive minor/major versions.
   - Any column addition must allow safe concurrent reads and writes by older application nodes before, during, and after migration execution.
   - For PostgreSQL, index creation on hot tables must support `CONCURRENTLY` without holding exclusive write locks on production data.

2. **Automated Migration Linter & Safety Gates (`tools/migrationguard`):**
   - Validate that every migration in `migrations/` has a complete `migration.yaml` metadata manifest specifying:
     - Canonical unique migration `id` matching directory name.
     - Target domain `module`.
     - Explicit DAG `dependencies`.
     - SHA-256 checksums for all dialect SQL scripts (`sqlite`, `postgres`).
     - Transactional flags per dialect (`transactional.sqlite`, `transactional.postgres`).
     - Explicit `lock_timeout` policy.
     - `disk_estimate` projection.
     - `compatibility` guarantee (e.g., backward/forward non-breaking expand).
     - `rollback` strategy and instructions.

3. **Multi-Version Coexistence Verification in Contract Suite:**
   - Execute contract tests (`tests/contract/migration_rollback_test.go` and `tests/contract/production_gates_certification_test.go`) validating:
     - Old binary narrow projections vs new binary expanded columns.
     - Preservation of unprojected fields during legacy record updates.
     - Online schema expansion concurrent with active read/write transactions.
     - Concurrency serialization and non-blocking index additions.

## Consequences

- **Positive:** Zero maintenance windows required for regular releases; elimination of schema drift and race conditions during blue/green deployments.
- **Positive:** Immediate build-time prevention of non-compliant migration scripts through `migrationguard`.
- **Operational:** Strict requirement that destructive "contract" migrations (dropping columns/tables) only occur in subsequent release cycles after all traffic to older binaries is fully retired.
