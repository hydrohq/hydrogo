# ADR-0226: Versioned field validation contract

- **Status:** Accepted
- **Date:** 2026-09-22
- **Related decisions:** [ADR-0210](0210-blueprint-scaffold-boundary.md), [ADR-0224](0224-global-resource-contract.md)

## Decision

Blueprint fields may declare provider-neutral validation metadata: length and
numeric bounds, a bounded regular expression, format hints (`email`, `date`,
`datetime`, `url`), uniqueness, immutability and a default value. The
generator carries this contract into the versioned Spec; it validates metadata
shape and rejects invalid ranges/formats/regular expressions before emitting
artifacts.

The contract is not an authorization policy and does not execute validation at
runtime by itself. Generated services, HTML forms and API transports must bind
the same metadata through typed validation adapters. Database uniqueness and
tenant scope remain explicit migration/service responsibilities.

## Verification

Blueprint tests cover strict decoding and mapping of validation metadata;
generator validation covers invalid bounds, formats and regular expressions;
generated `validation.go` is called by generated service Create/Update methods
and is compiled in the fixture suite. Runtime string formats and Unicode
length semantics are specified by [ADR-0257](0257-executable-generated-field-validation.md);
unique, immutable, and default semantics still require an Edition-owned
contract and must not be treated as automatically enforced.
