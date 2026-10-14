# ADR-0137: License Issuer and Runtime CLI Boundary

## Status

Accepted

## Context

The HydroGo runtime must support offline license verification where configured, but license generation capabilities and private signing keys must never be shipped in the customer distribution binary. Bundling `issue` and `keygen` commands inside the same binary increases the risk of inadvertent private key leaks and blurs operational boundaries.

## Decision

- `cmd/hydro-issuer` is the vendor-only license issuing binary carrying `keygen` and `issue` commands.
- `cmd/hydro` is the customer runtime CLI; it exposes only `license verify` and `license status`. Subcommands `issue` and `keygen` are rejected with explicit vendor-only errors.
- The runtime does not trust the `public_key` field embedded inside license files; it verifies signatures using the trusted public key injected into the official build.
- The signed `key_id` in claims is verified against the trusted key slot; rotation overlap windows are explicitly defined in release metadata.
- Private signing keys are maintained strictly in the issuer environment (CI/CD secret stores, access-controlled offline hardware).

## Consequences

- The client binary contains no signing capabilities, establishing an unambiguous boundary between issuer and runtime.
- Vendor release pipelines produce two distinct artifacts: the client runtime and the access-controlled issuer.
- Offline licenses have no centralized revocation server; risk is mitigated via bounded expiration, grace periods, and key rotation procedures.

## Verification

- `go test ./cmd/hydro ./cmd/hydro-issuer ./internal/licensing`
- Runtime tests confirm vendor-only commands are rejected and verify/status workflows succeed.
