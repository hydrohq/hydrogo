# R5 release manifest and signature validation

**Date:** 2026-09-08  
**Roadmap:** R5 release/reproducible build gate  
**Status:** Deterministic manifest, Ed25519 verify, and builder/toolchain provenance attestation contracts bound to the manifest subject succeeded; reproducible binary check with the same toolchain was added in CI. Production key ceremony, external provenance publishing, and cross-OS comparison remain open.

`internal/release` deterministically lists artifacts in the build root with path/size/SHA-256; it carries module, migration checksum, SDK/toolchain, and license inventory metadata. Artifacts named `.env`, `.key`, `.pem`, secret/credential are not included in the manifest. The manifest is signed with Ed25519 and Verify rejects any subsequent alteration.

```text
go test ./internal/release ./cmd/hydro
```

The signing key is not written into the runtime/config/manifest. For operational manifest verification, the `hydro release verify --manifest <path> --public-key <trusted-key-path>` CLI command was added and tested; the CLI does not trust the key within the manifest alone, it requires a match with an already distributed trusted Ed25519 public key. For provenance verification on the library side, `VerifyAttestationWithPublicKey` enforces the same trusted-key check; `VerifyAttestation`, which operates only with the embedded key, is kept as a self-contained signature check. CI builds the `hydro` binary twice with `-trimpath -buildvcs=false -ldflags=-buildid=` options and compares the SHA-256 values; the CI test now fails if the PostgreSQL contract environment is absent, and the Linux race job checks all `internal/...`, `modules/...`, and CLI packages with `-count=1`. `Attest` signs the manifest digest with the builder, source revision, and toolchain; `VerifyAttestation` rejects it if the manifest changes. Thus, binary reproducibility and subject integrity are proven on the same runner/toolchain. Production key ceremony, external provenance publishing, and Linux/Windows comparison remain open.
