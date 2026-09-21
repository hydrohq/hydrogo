# ADR-0208: Admin Extension origin-preserving sandbox handshake

- **Status:** Accepted; supersedes [ADR-0160](0160-admin-extension-runtime-host.md)
- **Date:** 2026-09-21
- **Related decisions:** [ADR-0195](0195-admin-extension-channel-binding.md), [ADR-0157](0157-admin-studio-extension-boundary.md)

## Context

ADR-0160 required an iframe sandbox without `allow-same-origin`. That setting
gives the framed document an opaque origin; browsers then report
`event.origin` as `"null"` for its `postMessage` events. ADR-0195 requires the
host to compare the exact signed manifest origin and `event.source` before
binding a channel, so the two requirements cannot work together.

## Decision

The extension iframe uses `sandbox="allow-scripts allow-same-origin
allow-forms allow-popups"`. Preserving the extension's own origin makes the
exact-origin handshake implementable. The iframe remains cross-origin from
HydroGo; it cannot read HydroGo DOM, cookies, CSRF tokens, or bearer tokens.
The host still enforces the signed manifest's HTTPS origin, CSP `frame-src`,
`event.source`, protocol version, single-use challenge, and server-side
channel authorization on every request.

`allow-same-origin` is therefore not a trust grant to HydroGo. It is a browser
compatibility requirement for the external origin that ADR-0195 already treats
as the trust-boundary input. Same-origin extension entrypoints remain rejected
by manifest validation and must never be deployed as a substitute for the
external-origin model.

## Consequences

- Browser MessageChannel binding can receive a non-opaque origin and proceed.
- The security boundary is explicit: origin/CSP/manifest validation plus
  server-side authorization, rather than an opaque sandbox origin.
- Signed external-extension browser E2E and staging rollout evidence remain
  required before commercial support claims.
