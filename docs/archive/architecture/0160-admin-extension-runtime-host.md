# ADR-0160: Signed Admin Extension runtime host

- Status: Superseded by [ADR-0208](0208-admin-extension-origin-handshake.md)
- Date: 2026-09-19
- Related decisions: [ADR-0157](0157-admin-studio-extension-boundary.md), [ADR-0159](0159-sqlite-external-write-cache-monitor.md)

## Context

The declarative Admin Studio contract is only useful to customers if a running
HydroGo process can discover approved manifests and present their surfaces.
Loading arbitrary files or Go plugins would allow untrusted code, credentials,
or routes to enter the Core process.

## Decision

HydroGo optionally loads signed JSON manifests from an operator-configured
directory. A separate operator-controlled JSON key map verifies Ed25519
signatures before any manifest is registered. Files are bounded, regular files
only, and duplicate/invalid manifests fail closed. Empty configuration keeps
the feature disabled with no route or file-watching overhead.

The protected `/admin/extensions/manifest` endpoint exposes validated metadata.
The protected `/admin/extensions/{extension_id}/{surface_id}` endpoint renders a
`no-store` iframe shell with a strict per-origin CSP, sandbox without
`allow-same-origin`, and a non-secret versioned bootstrap message. Cookies,
bearer tokens, database handles and arbitrary backend code are never passed to
the extension. Customer business logic remains an external webhook/event
consumer.

## Consequences

- Customers can deploy a signed Admin Studio UI without rebuilding or forking
  the HydroGo binary.
- Key rotation is explicit through the trusted key map; an artifact cannot
  introduce its own trust root.
- The shell is a transport boundary, not a claim that the extension host is a
  marketplace or that provider hosting is certified.
- CSP, origin, revocation, operator rollout, browser compatibility and staging
  E2E still require deployment evidence before a commercial support claim.
