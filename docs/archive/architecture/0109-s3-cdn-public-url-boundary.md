# ADR-0109: Explicit bounded S3/CDN public URL boundary

**Status:** Accepted  
**Date:** 2026-09-16

## Context

HydroGo already had an optional S3-compatible blob store, but its configured
`PublicBaseURL` and path-style option were not applied. Media transports need a
safe CDN URL without leaking provider details into domain code.

## Decision

`S3BlobStore` applies `ForcePathStyle` to the MinIO client and implements the
optional `PublicURLProvider`. Public URLs require an explicit base; HTTPS is
required except for loopback development endpoints. Object keys are bounded,
treated as path segments, and reject traversal, empty segments, backslashes,
and control characters. Core persistence continues to store opaque keys.

## Consequences

AWS S3, MinIO, R2, B2, and a CDN origin can be configured without changing
module contracts. A real provider, credentials, cache headers, invalidation,
and signed/private URL policy remain deployment-level gates.

## Verification

`go test ./internal/media -count=1` covers explicit URL configuration, safe
escaping, traversal rejection, and non-loopback HTTP rejection.
