# HydroGo SDK v1 support policy

This document defines the technical compatibility promise for the public
`extension/v1` contracts. It does not grant a commercial SLA or select a
distribution license; those require an explicit commercial decision.

## Versioning and support

- v1 follows semantic versioning. Additive fields and optional capabilities
  are backward compatible; existing fields and behavior do not change within
  v1 without a migration note.
- Each v1 minor release receives security fixes and correctness fixes for 12
  months after release. The latest minor is the supported feature line.
- The first release of each calendar year is an LTS line supported for 24
  months. LTS status is recorded in the release manifest and compatibility
  matrix; it is not a performance or availability guarantee.
- A v0 contract is experimental and receives a complete version pin,
  changelog and migration guide. It must not be consumed as a stable API.

## Deprecation

Deprecation requires a changelog entry, replacement API, generated-client
warning and migration guidance. A deprecated v1 symbol remains available for
at least two minor releases and six months, whichever is longer. Security
issues may require a shorter window with an explicit impact notice.

## Compatibility evidence

Every SDK release runs extension contract tests, generated frontend type
checks, two external-consumer fixtures and SQLite/PostgreSQL compatibility
checks. The release manifest records the SDK version, schema epoch and
generator version. An unsupported Core/Edition combination fails closed at
generation or startup rather than silently producing a partial project.

## Still requiring an explicit decision

Commercial license terms, paid support/SLA targets, response times, and
redistribution rights are intentionally not inferred by this technical policy.
They remain release gates in the professionalization task list.
