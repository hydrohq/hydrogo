# 0193. Release Provenance, SPDX SBOM & Production Readiness Certification

Date: 2026-09-20
Status: Accepted

## Context

Shipping production-grade releases of HydroGo across SaaS, Client Editions, and binary distributions demands verifiable supply chain security, complete provenance attestations, and compliance with the production gate checklist (`docs/validation/PRODUCTION-GATES.md`).

To satisfy enterprise procurement and compliance audits:
1. Every release build must generate a signed provenance manifest (`manifest.json`) cataloging artifact hashes, sizes, and Git commit bindings (`internal/release`).
2. Supply chain dependencies must be exported in deterministic SPDX 2.3 JSON Software Bill of Materials (SBOM) format (`tools/sbom`).
3. Release gate verification (`tools/releasegates`) must ensure that no release candidate is promoted to production while architectural proof gates or verification evidence remain open or uncertified.

## Decision

1. **Cryptographic Release Provenance & Attestation (`internal/release`):**
   - Bind every built binary, template bundle, and migration asset to SHA-256 digests.
   - Enforce automatic secret leakage detection, rejecting artifacts matching credential, private key, or environment variable patterns.
   - Rejection of mutable symlink artifacts or path traversals in release directories.

2. **Automated SPDX SBOM Generation (`tools/sbom`):**
   - Generate reproducible, deterministic SPDX-compliant SBOM metadata indexing Go modules, direct dependencies, licenses, and module checksums.

3. **Production Gate Certification & Readiness Sign-off (`tools/releasegates`):**
   - Enforce mandatory validation against `docs/validation/PRODUCTION-GATES.md`.
   - Reject release promotion if required ADRs, certified database contract tests, or validation reports are missing.

## Consequences

- **Positive:** Complete supply chain transparency, reproducible release verification, and compliance with enterprise security standards.
- **Positive:** Prevention of accidental secret leaks or untracked dependency injection into customer binaries.
- **Operational:** Release CI pipeline automatically builds and verifies SBOM and provenance attestations before artifact signing and publishing.
