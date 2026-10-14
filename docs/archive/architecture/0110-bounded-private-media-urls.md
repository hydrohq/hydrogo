# ADR-0110: Bounded private media presigned URLs

**Status:** Accepted  
**Date:** 2026-09-16

## Context

Private media served through an application proxy is useful for local and
small deployments, but S3-compatible deployments need an optional short-lived
URL capability. A permanent or unvalidated URL would bypass the visibility
and key boundaries.

## Decision

`S3BlobStore` implements the optional `SignedURLProvider` with
`PresignedURL`. Callers must authorize first; expiry is bounded to 1 second
through 24 hours and object keys use the same traversal-safe validation as
public URLs. The capability is not enabled implicitly and does not change
asset visibility.

## Consequences

Client Editions can hand private media to a trusted CDN/object store without
leaking credentials or provider details into domain code. Real provider
connectivity, cache headers, revocation behavior, and CDN policy remain
deployment evidence gates.

## Verification

`go test ./internal/media -count=1` covers expiry bounds and key validation;
live presign/provider behavior is intentionally reserved for the external S3
operator gate.
