# ADR-0107: Bounded provider-neutral async relation contract

**Status:** Accepted  
**Date:** 2026-09-16

## Context

Admin relation fields currently support static options, while Payload-style
large collections need asynchronous search. Without a shared envelope, clients
can issue unbounded queries or receive arbitrary record data.

## Decision

Admin Studio exposes `RelationQuery`, `ValidateRelationQuery`,
`RelationOption`, and `NormalizeRelationOptions`. Collection, term, cursor, and
limit are bounded; the default is 50 and the hard maximum is 100. Options carry
only ID and label, reject duplicates, and are sorted deterministically. Data
lookup, authorization, cursor signing, and caching remain module/provider-owned.

## Consequences

Relation pickers can be implemented consistently across design systems without
forcing a search provider or extra service into Compact deployments. The server
must revalidate every request; client-side bounds are only an ergonomic aid.

## Verification

`go test ./internal/adminstudio -count=1` covers bounds, normalization, and
duplicate rejection.
