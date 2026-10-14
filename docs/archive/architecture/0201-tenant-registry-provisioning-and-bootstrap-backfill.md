# ADR-0201: Tenant registry, idempotent provisioning, and bootstrap backfill

- Status: Accepted
- Date: 2026-09-21
- Supersedes: [ADR-0186](0186-tenant-provisioning-and-backfill-cli.md) for new tenant lifecycle management
- Related decisions: [ADR-0149](0149-shared-multi-tenant-saas-evolution.md), [ADR-0154](0154-tenant-scope-primitive.md), [ADR-0180](0180-postgres-tenant-rls-enforcement.md)

## Context

An operator needs a restart-safe path to initialize a shared-database tenant without exposing a public sign-up, payment, or Commerce flow. The former legacy backfill command can assign empty namespaces but cannot establish a tenant lifecycle, baseline configuration, first administrator, or system-level operator evidence.

## Decision

`hydro tenant provision --tenant-id <id> --admin-email <email> --admin-password-stdin` is the sole initial provisioning entry point. It requires `tenancy.enabled=true`, a current migration ledger, and PostgreSQL RLS preflight where PostgreSQL is selected. Passwords are accepted only over standard input and are not written to reports or audit records.

The command records `tenant.provision.started`, performs one short transaction that creates/resumes the registry row, applies idempotent site/locale/timezone/media-folder defaults, creates the initial `admin` account when it does not already exist, and activates the tenant. After the committed transaction it emits individual system-ledger receipts for registry, defaults, administrator create/reuse, and activation, then `tenant.provision.completed`; failures receive `tenant.provision.failed`. These events live in the global `tenant_system_audit` ledger rather than tenant content/audit data.

`hydro tenant backfill --active [--dry-run|--confirm]` walks only active registry tenants and applies missing bootstrap defaults in independent short transactions. It creates an append-only `tenant.backfill.applied` event for every executed tenant. The prior `--tenant-id` form remains the explicit legacy empty-namespace migration tool; it is not the provisioning mechanism. Commerce is deferred and has no bootstrap data or active-tenant backfill responsibility.

## Consequences

- Provisioning is repeatable after interruption and does not reset existing passwords or tenant data.
- The Compact profile starts no service: these are explicit operator commands. Larger profiles avoid a global backfill lock because each tenant is handled independently.
- The registry migration is expand-only, so old binaries ignore it and new binaries refuse provisioning against an incomplete ledger. Blue/green rollout order is migrate, deploy the new binary, then provision/backfill.
- PostgreSQL RLS is verified as a precondition, but a live PostgreSQL isolation and recovery drill remains a production release gate; SQLite cannot supply database-side RLS evidence.
