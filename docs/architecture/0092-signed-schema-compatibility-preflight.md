# ADR-0092: Signed schema compatibility preflight

## Status

Proposed

## Context

Runtimes distributed via the SDK share the same migration history during a blue/green
cutover. Exposing raw SQL or database dialect internals to the public CLI violates both the
information hierarchy and the Core boundary.

## Decision

`hydro schema diff` compares the migration checksum ledgers of two signed release manifests.
No comparison is executed without first passing trusted public key verification. New migration
IDs are reported additively; removing an existing ID or altering its checksum is treated as an
incompatibility error. The actual execution of migrations is not the responsibility of this command.

## Consequences

Release and blue/green preflights become deterministic and independent of database SQL dialects.
Manifest generation and signing must take place within the CI/release environment; empty or corrupted
ledger metadata fails closed by design.

## References

- [ADR-0091: SDK runtime artifact distribution](0091-sdk-runtime-artifact-distribution.md)
- [ADR-0060: Migration metadata lint](0060-migration-metadata-lint.md)
