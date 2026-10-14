# ADR-0254: Verify generated resource specs against the requested source

- **Status:** Accepted
- **Date:** 2026-09-23
- **Related decisions:** [ADR-0253](0253-generated-resource-contract-version-lock.md), [ADR-0241](0241-versioned-blueprint-input.md)

## Context

Artifact checksums prove that files match a lock, but do not prove that the
locked `spec.json` represents the resource source an operator asked to check.
If a generated spec and its artifact digest were both edited, a source-driven
compatibility check could otherwise accept a different generated contract.

## Decision

Before reporting compatibility, the checker loads and validates the generated
`spec.json`, verifies that its explicit contract version matches
`generator.lock`, and compares its normalized resource contract with the
requested source `Spec`. It remains read-only and rejects symlinked generated
paths/files using the existing path guards. Artifact checksums continue to
verify generated outputs independently.

## Consequences

- A changed embedded resource contract cannot be blessed by updating only its
  artifact checksum.
- Existing v0 specs and pre-version-marker locks retain the compatibility
  behavior defined in ADR-0253.
- This proves source/generated consistency, not that the Edition's runtime
  policy, database RLS, or migration behavior is correct.

## Verification

`TestCheckCompatibilityRejectsEmbeddedSpecDriftWithUpdatedArtifactDigest`
modifies generated `spec.json`, updates its artifact checksum, and verifies the
checker still rejects the mismatch with the original source contract.
