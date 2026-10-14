# ADR-0195: Admin Extension Channel Binding Protocol v1

- **Status:** Accepted (protocol contract; browser E2E open)
- **Date:** 2026-09-20
- **Related decisions:** [ADR-0157](0157-admin-studio-extension-boundary.md), [ADR-0160](0160-admin-extension-runtime-host.md), [ADR-0192](0192-extension-sdk-boundary-admin-studio-security.md)

## Context

Admin Extension surfaces run outside the HydroGo origin. The browser transport
must therefore bind an authenticated Admin session, one verified manifest
surface, and one concrete iframe without treating browser messages as an
authority source.

## Decision

The host performs a short-lived `window.postMessage` handshake and validates
`event.source`, the exact manifest entrypoint origin, protocol version,
extension/surface identifiers, and a single-use random challenge. Only after
that validation may the host transfer a `MessagePort` and use it for subsequent
messages.

`MessagePort` is a transport mechanism, not an authorization mechanism. Every
request is resolved against the server-side `ChannelContext`, the current Admin
session, tenant policy, operator grant, and versioned capability contract.

Channels are revoked on logout, tenant or permission changes, iframe unload,
expiry, or explicit operator suspension. A `revoke` message is a UX signal;
server-side authorization is still rechecked for every request.

Replay protection uses a bounded channel-local request-ID cache. Business
mutations additionally require durable application-level idempotency and must
not rely on the in-memory replay cache.

The external-origin rule is explicit: HydroGo does not require the extension
to share HydroGo's origin. Each manifest has one HTTPS entrypoint origin, and
all surface paths must remain relative to that entrypoint. Production HTTP and
localhost entrypoints are rejected; local development uses a separate
development proxy flow.

## Consequences

- The first message remains origin/source-bound even though later traffic uses
  an isolated `MessagePort`.
- A compromised extension can exercise only the capabilities already granted
  to its own channel; it cannot assert another actor, tenant, extension, or
  capability through client-supplied fields.
- Browser compatibility and staging E2E tests are required before a commercial
  support claim.
