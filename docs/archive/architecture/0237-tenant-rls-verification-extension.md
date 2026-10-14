# ADR-0237: Edition-extensible tenant RLS verification

- **Status:** Accepted
- **Date:** 2026-09-22
- **Related decisions:** [ADR-0180](0180-postgres-tenant-rls-enforcement.md), [ADR-0233](0233-public-tenant-scope-contract.md), [ADR-0235](0235-generated-tenant-scoped-resource-contract.md)

## Decision

Tenant provisioning exposes a table-parameterized RLS verifier in addition to
the legacy Core-table wrapper. Client Editions must pass every generated
tenant-scoped table through the same fail-closed check before enabling shared
mode or provisioning a tenant. Table identifiers are bounded, validated and
deduplicated before any PostgreSQL metadata query; SQLite validates the same
input but relies on explicit repository predicates.

The verifier confirms row-level security, forced row-level security and the
canonical `hydrogo_tenant_isolation` policy in the current schema. It does not
create or alter policies and therefore preserves the no-runtime-DDL rule.

## Consequences

- Generated tables cannot be silently omitted from the Edition's RLS gate.
- Existing Core callers retain the default four-table behavior.
- Policy creation, migration review and database ownership remain operator-
  controlled; a successful verifier is necessary but not a complete restore or
  capacity certification.

## Verification

`internal/tenant/provision_test.go` validates accepted SQLite table sets and
rejects duplicates, unsafe identifiers and empty sets. Generated tenant
runtime wiring now records the verifier as a release checklist item, while
PostgreSQL integration continues to verify the actual catalog policy state.
