# ADR-0233: Public tenant scope contract

- **Status:** Accepted
- **Date:** 2026-09-22
- **Related decisions:** [ADR-0228](0228-public-resource-query-contract.md), [ADR-0232](0232-bounded-offline-sync-contract.md)

## Decision

Client Editions use the public `extension/v1` tenant scope contract at the
request, worker and cache boundaries. A scope is validated before entering a
context or cache key; missing or malformed scope fails closed. PostgreSQL
adapters may bind the validated scope to a transaction-local RLS setting, while
SQLite adapters must apply the same explicit predicate in repositories.

The contract does not resolve a tenant from an untrusted header or URL. An
Edition-owned authenticated resolver must establish the scope, and generated
resource policies remain responsible for membership, role and cross-tenant
administration rules.

## Verification

Public tests cover missing scope, invalid components and namespaced cache keys.
Tenant-scoped resource implementations must certify RLS/predicate behavior,
background-job propagation and blue/green session continuity on both adapters.
