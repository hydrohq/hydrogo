# ADR-0093: Permission contract preflight

## Status

Proposed

## Context

Generated resources must not silently introduce ambiguous or unsafe capability
names. Permission metadata is part of the public contract, while authorization
evaluation remains in the Core/module boundary.

## Decision

Resource specs accept bounded lowercase permission identifiers using the
`resource.action` convention (with `._:-` separators), reject duplicates,
traversal-like sequences and whitespace, and expose an offline
`hydro permissions check --spec SPEC` command. The command validates the full
spec and reports only collection/count information.

## Consequences

Generators fail early and deterministically; no database or external identity
provider is needed for authoring checks. This does not grant permissions or
replace runtime policy tests.

## References

- [ADR-0027: Provider-neutral field registry](0027-provider-neutral-field-registry.md)
- [ADR-0013: Identity and security boundaries](0013-identity-and-security-boundaries.md)
