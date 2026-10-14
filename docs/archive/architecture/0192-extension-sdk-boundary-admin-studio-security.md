# 0192. Extension SDK Boundary & Admin Studio Security Certification

Date: 2026-09-20
Status: Accepted

## Context

HydroGo allows extensibility via third-party extensions, custom blocks, admin UI surfaces, and SDK capability kits without compromising the security or stability of HydroCore.

To maintain strict boundaries between core logic and untrusted or customer-developed extensions:
1. Extensions communicate with the core host only through versioned, typed SDK contracts (`extension/v1`).
2. Admin Studio UI plugins must provide cryptographic signature verification and manifest validation (`internal/adminstudio`), ensuring runtime isolation, same-origin target resolution, and protection against UI redressing / clickjacking / open redirects.
3. Import linting (`tools/importguard`) and action pinning (`tools/actionguard`) prevent architecture erosion, preventing modules or extensions from importing forbidden concrete database adapters or SQL driver internals.

## Decision

1. **SDK Contract & Versioning Discipline (`extension/v1`):**
   - Formalize public capability kits, lifecycle contracts, theme bundles, and form/search schemas under `extension/v1`.
   - Enforce semantic major version compatibility rules, preventing breaking changes across minor SDK increments.

2. **Cryptographic Manifest Verification & Admin Studio Security:**
   - Require signed extension manifests with ed25519/HMAC key verification.
   - Enforce same-origin relative URLs on all extension iframe / UI surfaces, failing closed on external domain injection attempts.

3. **Automated Architectural Guardrails:**
   - Execute `importguard` on every CI build to block leaks of dialect SQL drivers or storage internals into domain and service layers.
   - Execute `actionguard` to ensure all external CI workflows remain strictly pinned to immutable commit hashes.

## Consequences

- **Positive:** Third-party extensions cannot breach core security or leak sensitive tenant memory contexts.
- **Positive:** Complete elimination of arbitrary JavaScript execution in privileged Admin Studio frames.
- **Operational:** Extension developers must package plugins with valid manifest signatures conforming to `extension/v1`.
