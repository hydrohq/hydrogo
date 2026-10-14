# ADR-0034: Module-owned admin query allowlists

## Status

Accepted

## Decision

Before a `ListQuery` reaches a repository, the owning module constrains sort
and filter names against explicit allowlists. Unknown fields are discarded and
descending order is represented only by a leading `-` on an allowed field.

## Compatibility and capacity

This is additive, dialect-neutral, and requires no migration. Allowlist checks
are in-memory and linear in supplied fields; they add no cache, worker, or
connection-pool demand.
