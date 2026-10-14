# ADR-0015: Versioned HTTP API paths

- **Status:** Accepted
- **Date:** 2026-09-09
- **Supersedes:** none
- **Implementation status:** `/api/v1` aliases are implemented for Pages, News, and Commerce; existing `/api` paths remain compatibility surfaces and advertise their successor.

## Context

The implementation plan defines the JSON API as versioned under `/api/v1`, while the first module routes were shipped under `/api`. Silent path changes would break existing clients, but leaving only unversioned paths would make future compatibility guarantees ambiguous.

## Decision

- New and documented JSON API clients use `/api/v1/...`.
- Existing `/api/...` routes remain available as compatibility aliases.
- Legacy aliases send `Deprecation: true` and a `Link` successor-version header.
- Both paths use the same handler, authorization, CSRF, validation, and service policy.
- Commerce payment webhooks also expose a versioned `/api/v1/webhooks/payment/{provider}` alias alongside the legacy path.
- A future removal of `/api/...` requires a separately announced migration window and release decision; this ADR does not set a sunset date.

## Consequences

The router owns the compatibility mapping, so modules do not duplicate route declarations or drift between versions. The alias is intentionally limited to paths beginning with `/api/`; admin, public HTML, media, and operational paths are unaffected. Contract tests must cover both the versioned and compatibility forms for new public API routes.
