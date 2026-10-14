# ADR-0028: Core Isolation and Licensing Architecture

## Status
Superseded by [ADR-0137](0137-commercial-license-issuer-boundary.md) for the issuer/runtime CLI boundary. The core isolation and offline verification decisions remain valid.

## Context

When distributing the HydroGo platform to corporate clients and agencies, the core engine logic (**HydroGo Core**) must remain isolated from client-specific themes, templates, migrations, and configuration files (**Client Edition**).

Generated client projects should not contain HydroGo Core source code; client projects should be freely manageable in client repositories while Core remains independent.

Three distribution models were evaluated:
1. **Private Go Module:** Consumed via access-controlled Git / GOPROXY.
2. **Standalone HydroGo Binary:** Precompiled single executable capable of loading configuration and themes.
3. **Hosted / SaaS Runtime:** Managed runtime environment operated in cloud clusters.

## Decision

1. **Core Isolation:** Generated client projects do not include HydroGo Core source code. The project generator produces only the Client Edition (`extension/v1`), themes (`themes/`), configuration (`config.yaml`), migrations, frontends, and E2E tests.
2. **Hybrid Distribution:** Core is distributed as a standalone binary for standard deployments or as a Go module for custom corporate projects.
3. **Lock File (`hydrogo.lock`):** The `hydrogo.lock` file in the root of generated projects pins the Core version, generator version, edition tier, preset, project ID, and enabled capability set.
4. **Ed25519-Signed Offline Licensing:**
   - The optional license file (`hydro.lic`) is signed offline using Ed25519.
   - The Public Key required for verification is embedded or configured; the Private Key is never shipped to client environments.
   - Verification covers `customer`, `project_id`, `edition`, `features`, `expires_at`, and `grace_period_days`.
   - When a license expires, a configurable **Grace Period** activates to prevent sudden operational disruptions.
5. **CLI Separation:** Vendor-only `hydro-issuer` handles `issue` and `keygen`; client `cmd/hydro` provides `verify` and `status`. See [ADR-0137](0137-commercial-license-issuer-boundary.md).
6. **Project Scaffold:** `hydro new PROJECT --preset=...` creates a client-owned scaffold with no Core source. The scaffold contains config, lock metadata, a minimal theme, project migration/custom placeholders, and E2E manifests.
7. **Runtime Enforcement:** When `license.required: true` is configured, `serve` requires `hydro.lic`, matching project/edition/features, and the trusted public key. Development builds without an injected vendor key fail closed for licensed projects (or run in dev mode via `--dev` / `HYDRO_DEV_MODE=1`).

## Resource & Security Boundaries

- **Cryptographic Boundaries:** Ed25519 signatures detect tampering deterministically. As with any client-side binary, reverse engineering is an inherent reality; technical signatures are combined with contracts and auditability.
- **Resource Footprint:** License verification executes once at boot in memory with zero phone-home network requests, incurring negligible CPU and memory overhead in the Compact profile.
- **Key Trust:** The `public_key` field in a license document is informational only. Runtime verification requires trusted keys injected into the build; accepting arbitrary self-advertised keys is rejected.

## Consequences

- Client projects are clean, lightweight, and managed independently of Core repositories.
- Air-gapped and enterprise on-premise environments operate without external network dependencies.
