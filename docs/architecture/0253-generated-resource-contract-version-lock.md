# ADR-0253: Persist the resource contract version in generator locks

- **Status:** Accepted
- **Date:** 2026-09-22
- **Related decisions:** [ADR-0241](0241-versioned-blueprint-input.md), [ADR-0211](0211-declarative-module-resource-contract.md)

## Context

Blueprint and generated `spec.json` documents already have an explicit `v0`
format boundary. The generated `generator.lock`, however, recorded only the
generator version and a normalized resource-spec digest. Operators and
compatibility checks could not identify the persisted resource-contract
version directly from the lock.

## Decision

New generator locks record `contract_version` using the same explicit version
as the generated resource document. The current supported value is `v0`.
Compatibility checks reject a non-empty unsupported value before declaring a
generated resource compatible. A missing value in a pre-existing lock is
treated as `v0` for backward compatibility; it is not silently upgraded or
rewritten by a read-only compatibility check.

The version is metadata, not permission to activate routes, execute DDL, infer
authorization, or overwrite human-owned `custom/` files. Introducing another
resource contract version still requires an ADR, loader/schema support,
conversion/compatibility tests, and regenerated reviewable artifacts.

## Consequences

- Release tooling can identify both generator implementation and resource
  contract format.
- Existing lockfiles remain checkable without mutation.
- The resource contract remains explicitly v0; this decision does not claim
  v1 semantics or customer compatibility certification.

## Verification

`TestCheckCompatibility` verifies the new lock marker, legacy missing-marker
compatibility, and fail-closed rejection of an unknown contract version.
