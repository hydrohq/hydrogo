# R5 reproducible release validation — 2026-09-14

- `scripts/verify-production-gates.ps1` and `.sh` build two normalized binaries
  and fail when SHA-256 digests differ.
- `.github/workflows/ci.yml` repeats the same comparison on CI.
- Local verification passes the normalized build path and deterministic SBOM.
- Signing keys, registry attestations, and retention policy remain external
  gates and are not represented as complete.
