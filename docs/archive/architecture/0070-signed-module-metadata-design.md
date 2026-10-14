# ADR-0070: Design signed module metadata without a marketplace

## Status

Accepted

## Decision

HydroGo defines a future package metadata shape and fail-closed verification
rules, but does not add a marketplace, runtime Go plugin loader, or implicit
network dependency. The release verifier remains the trust boundary and uses a
deployment-provided trusted public key plus an explicit module allowlist.

## Consequences

Module packages can be reviewed and attested consistently across editions.
Signing key ceremony, registry selection, rotation, and licensing decisions
must be completed before treating package distribution as production-ready.
