# ADR-0143: Binary-only customer distribution

- **Status:** Accepted

## Decision

Customer installations use an allowlisted package containing the stripped (optionally obfuscated) `hydro` runtime, `hydro.lic`, `hydrogo.lock`, deployment configuration, themes, migrations, and Client Edition files. Core/module source, `cmd`, `.git`, Go module metadata, issuer, and private keys are rejected by the package builder.

The vendor build injects one or more trusted public keys through protected CI variables. `HYDRO_OBFUSCATE=true` requires the vendor-controlled Garble tool; it is not enabled implicitly because reproducibility and toolchain provenance must remain explicit.

## Consequences

- Customers can install on their own servers without receiving HydroGo source.
- The final Docker/runtime artifact contains no build-stage source files.
- A privileged customer can still inspect or patch a running binary; this design reduces casual reuse and source disclosure, not all reverse engineering.

## Validation

Run `scripts/build-customer-runtime.ps1` or `.sh` with `HYDRO_CUSTOMER_LICENSE` and `HYDRO_TRUSTED_PUBLIC_KEYS`. The builder fails closed on missing keys/license or forbidden files and emits `PACKAGE-MANIFEST.md`.
