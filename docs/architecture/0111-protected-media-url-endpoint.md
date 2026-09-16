# ADR-0111: Protected endpoint for private media URLs

**Status:** Accepted  
**Date:** 2026-09-16

## Context

S3 presigned URL generation was available as a storage capability, but no
standard transport endpoint exposed it to an authorized editor or integration.
Exposing raw object keys or making the endpoint public would weaken media
visibility boundaries.

## Decision

Media transport exposes protected `GET /api/media/{id}/url`. It requires an
active editor/admin, a ready asset, an explicitly configured signer, and a
bounded optional `expires` value (default 5 minutes, maximum 24 hours). Errors
return generic operational messages; the response contains only asset ID, URL,
and seconds until expiry.

## Consequences

Client Editions can hand private media to a trusted CDN without embedding
credentials. Authorization, audit requirements, URL revocation, and provider
connectivity remain deployment/module responsibilities.

## Verification

`go test ./internal/media -count=1` covers anonymous denial, successful editor
response, expiry bounds, and missing-provider failure.
