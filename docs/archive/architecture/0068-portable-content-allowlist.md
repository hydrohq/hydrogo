# ADR-0068: Keep portable export explicitly allowlisted

## Status

Accepted

## Decision

Portable documents contain only the versioned content table allowlist. Locale,
revision, relation, and media metadata are preserved, while identity,
session, audit, outbox, and job state never cross the boundary.

## Consequences

Import cannot silently restore authentication or operational state. New tables
require an explicit format-version decision and contract fixtures.
