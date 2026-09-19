# ADR-0157: Runtime Admin Studio extension boundary

- Status: Accepted (declarative external-surface foundation)
- Date: 2026-09-19
- Related decisions: [ADR-0008](0008-module-and-extension-contract.md), [ADR-0020](0020-headless-first-admin-studio-and-frontend-templates.md), [ADR-0013](0013-identity-and-security-boundaries.md)

## Context

Templ views are compiled into the HydroGo binary. That is desirable for the
first-party Admin Studio, but it cannot be the customer customization contract:
customers must add reports, settings and workflow screens without a Core fork
or a new HydroGo build. Go's runtime `plugin` package is not a safe portable
answer because its ABI/toolchain coupling and Windows support are poor, and it
would execute untrusted customer code in the Core process.

## Decision

Use two explicit extension planes:

1. **Declarative Admin surface:** a customer-owned, signed JSON manifest is
   loaded at startup/deployment time and validated by
   `extension/v1.AdminExtensionManifest`. It contributes navigation/surface
   metadata and HTTPS entrypoint paths only. The host renders each surface in
   an isolated external origin (normally a sandboxed iframe) and passes only
   versioned, permission-scoped postMessage data. `internal/adminstudio` keeps
   only validated manifests; it never loads a Go plugin or executes customer
   code.
2. **Backend integration:** business rules run outside Core through the existing
   versioned webhook/event boundary. `AdminSubscription` declares event names;
   delivery, retries, signatures, idempotency and operator quarantine stay in
   the webhook gateway. No customer code is dynamically linked into HydroGo.

The manifest validator requires an absolute HTTPS entrypoint without
credentials, same-origin relative surface paths, bounded identifiers, explicit
permissions and deduplicated event subscriptions. The host still enforces
authorization; manifest permissions are not grants.

## Consequences

- Agencies can ship an independently deployed admin report/settings UI and
  backend integration without rebuilding or forking the closed binary.
- Core remains portable across Windows/Linux and keeps a small, auditable trust
  boundary.
- The first-party Templ Admin Studio remains compiled and fast; extension
  surfaces are an explicit external-origin product capability.
- Signature verification, iframe CSP/COOP policy, postMessage protocol,
  extension lifecycle/rollback and webhook provider certification are required
  before advertising this as a production marketplace.
