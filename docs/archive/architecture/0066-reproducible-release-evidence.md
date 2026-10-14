# ADR-0066: Require reproducible build evidence without claiming provenance

## Status

Accepted

## Decision

CI and production-gate scripts compare two normalized HydroGo build digests.
Release evidence records the inputs and SBOM digest. Cryptographic signing,
registry attestations, and retention are not enabled implicitly; they require
an external key and deployment decision.

## Consequences

Non-deterministic build changes fail before release evidence is accepted.
Operators can add an attestation backend later without changing Core contracts.
