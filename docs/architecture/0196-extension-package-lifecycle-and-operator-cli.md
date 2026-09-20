# ADR-0196: Extension Package Lifecycle and Operator CLI

- **Status:** Accepted (artifact contract and lifecycle foundation; CLI E2E open)
- **Date:** 2026-09-20
- **Related decisions:** [ADR-0070](0070-signed-module-metadata-design.md), [ADR-0137](0137-commercial-license-issuer-boundary.md), [ADR-0143](0143-binary-only-customer-distribution.md), [ADR-0195](0195-admin-extension-channel-binding.md)

## Context

Customer Admin Extensions must be installable without loading Go plugins or
executing customer code inside HydroGo. Installation must not silently grant
capabilities or activate routes, webhook delivery, or extension surfaces.
Vendor signing authority must remain outside the customer runtime.

## Decision

The vendor-only `hydro-issuer` validates and signs an extension package. The
signature covers canonical manifest metadata and a deterministic, sorted flat
hash list for every declared package asset. A package is not accepted when the
manifest, signature, key ID, asset path, asset size, or asset digest is invalid.

The customer runtime implements a staging-first lifecycle:

```text
discovered → verified → installed → approved → enabled
                                      ↘ suspended
enabled → suspended → enabled
any trusted state → revoked → removed
```

`install` only stores a verified artifact in a staging area. `approve` records
operator grants separately from the manifest's requested capabilities. `enable`
is the only transition that registers validated surfaces and subscriptions.
`suspend` preserves approval but stops runtime activity. `revoke` invalidates
trust and cannot be undone by re-enabling the same artifact.

Bulk approval is supported, but `--all-requested` is an explicit operator
choice and never the default. Interactive approval must display every requested
capability and event subscription before confirmation.

No marketplace or implicit network dependency is introduced. Key rotation,
revocation, rollback, and lifecycle transitions are explicit and auditable.

## Consequences

- Loading an unapproved package creates no route, worker, webhook delivery, or
  executable runtime code.
- A package tamper is detected even when only a non-manifest asset changes.
- The vendor private key never enters the customer package or runtime.
- The artifact format, staging directory, operator grant store, and CLI state
  transitions require integration tests before production support.
