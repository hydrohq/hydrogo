# Hydro SDK and Private HydroGo Runtime Distribution Plan

**Status:** Contracts and secure local artifact pipeline implemented; broker,
authentication, signed releases, and installer distributions remain open gates. Detailed status:
[`IMPLEMENTATION_STATUS.md`](IMPLEMENTATION_STATUS.md).  
**Objective:** Provide a Node.js-style installable public `hydro` CLI across Linux, Windows,
and macOS that orchestrates the private HydroGo Core runtime without distributing source code.

## Core Architectural Ruling

The public SDK encapsulates the CLI, generators, starter templates, contracts, frontend adapters,
and developer tooling. HydroGo Core and proprietary commercial modules remain private; source code
is never copied to clients. The runtime is delivered as signed binaries and artifacts via an authenticated channel.

This approach aligns with Core isolation (ADR-0028), `hydrogo.lock`, edition manifests, and offline
licensing decisions. The SDK does not treat Go package boundaries or `internal` package visibility as a
security sandbox; trusted code executing within the same binary is treated as trusted.

## Developer Workflow

```text
hydro login
hydro project new haber-sitesi --preset=news
cd haber-sitesi
hydro run
hydro resource new articles
hydro generate
hydro migrate
```

## Distribution Architecture

### Public Hydro SDK

- `hydro` CLI
- Project and resource generators
- Starter templates
- OpenAPI and TypeScript client generators
- React, HTMX, and vanilla JavaScript adapters
- `doctor`, `check`, `schema diff`, `api diff`
- Local development proxy and test runner

### Private Artifact Service

- Platform-specific runtime binaries
- Module and kit artifacts
- Signed release manifests
- Short-lived download URLs and tokens
- License and edition entitlement verification

### HydroGo Runtime

- Core engine + selected modules
- Database migration and compatibility metadata
- Binary and manifest signature verification against trusted public keys
- Offline `hydro.lic` verification

## Private GitHub Access Boundaries

`git clone` and `git ls-remote` are strictly excluded from the primary end-user authorization mechanism.
Git commands may only be used for diagnostic purposes in internal corporate developer workflows. Repository
access does not equal an active license, edition entitlement, or artifact integrity guarantee. Furthermore,
it exposes risks around token exfiltration, SSO failures, proxy misconfigurations, rate limits, and ambiguous error handling.

Proposed distribution flow:

```text
hydro login
    ↓
Hydro account / entitlement service
    ↓
GitHub App or private artifact registry
    ↓
Short-lived signed download URL
    ↓
Checksum and signature verification
    ↓
Local runtime cache
```

Short-lived GitHub App installation tokens or server-side credentials are used exclusively on the server;
persistent GitHub PATs are never written to client machines. Releases pull precompiled CI artifacts rather
than source repositories.

## Operating System Installers

- Windows: MSI installer, WinGet manifest, Authenticode-signed `.exe`, Windows Credential Manager integration.
- macOS: Homebrew tap, Apple-notarized universal/arm64/amd64 binaries, macOS Keychain integration.
- Linux: `.deb`, `.rpm`, tarballs; amd64/arm64 architectures; strict checksum and GPG signature verification.
- Post-install: `hydro doctor` verifies PATH, platform, runtime, license status, and cryptographic signatures.

## Lockfiles and Compatibility

Projects lock SDK versions, runtime revisions, contracts, editions, module sets, and cryptographic hashes in `hydrogo.lock`.
The CLI enforces:

- SDK, runtime, and module compatibility
- License and edition entitlements
- Platform and architecture compatibility
- Database migration and API schema compatibility
- Cryptographic signatures and SHA-256 checksums
- Offline grace periods

## Security Safeguards

- Private signing keys reside strictly within hardened release/signing environments.
- Private repository access tokens are never saved to client filesystems.
- Artifact download URLs are short-lived.
- Binaries and release manifests are verified against trusted keys before execution.
- Download, authentication, and license logs strictly redact secrets and tokens.
- Offline and air-gapped environments are supported via signed offline licenses and manual artifact bundles.
- License expiration does not abruptly crash public sites; entitlement and update lifecycles are explicitly defined.

## Execution Sequence

1. Finalize SDK, runtime, and contract versioning models and lockfile formats.
2. Produce signed, cross-platform runtime artifacts in CI.
3. Integrate artifact manifest and checksum verification into the CLI.
4. Implement OAuth 2.1 / device code authentication and token exchange in `hydro login`.
5. Integrate GitHub App / private registry access behind the broker service.
6. Publish official Windows, macOS, and Linux installers.
7. Verify offline licensing and air-gapped runbooks.
8. Rehearse key revocation, rotation, mirroring, and rollback drills.

## Open Decisions

- Hosted Hydro account system versus GitHub organization identity?
- Primary artifact registry: GitHub Releases, GHCR, or dedicated S3-compatible object storage?
- Separation of commercial entitlements from repository membership?
- Artifact update protocol for fully air-gapped customer installations?
- Monolithic runtime binary versus modular plugin bundles?

Until these decisions are formally accepted, public SDK releases and customer distribution commitments remain open.
