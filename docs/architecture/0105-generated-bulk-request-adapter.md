# ADR-0105: Mirror the Admin bulk envelope in generated TypeScript

**Status:** Accepted  
**Date:** 2026-09-16

## Context

The Go Admin Studio bulk contract bounds and normalizes selections, but a
frontend generated from the same resource could still send unsorted IDs,
duplicates, or malformed action names.

## Decision

The generated `bulk.ts` adapter exposes a resource-specific bulk request type
and builder that applies the same 500-record, action, ID, duplicate, and control
character checks before returning a deterministic sorted envelope. It remains
dependency-free and does not perform authorization or network mutation.

## Consequences

Frontend and backend callers share one predictable request shape, reducing
avoidable validation failures while preserving module-owned policy and
transaction semantics. The Go contract remains authoritative for server-side
security; client validation is an ergonomic early check only.

## Verification

`go test ./internal/generator -count=1` verifies the generated adapter contains
the request builder and normalization guards.
