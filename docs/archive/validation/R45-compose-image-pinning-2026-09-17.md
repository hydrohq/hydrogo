# R45 — Local Compose image tag pinning

**Date:** 2026-09-17  
**Status:** Successful (static configuration verification)

MinIO and Redis images in the local storage Compose fixture use release tags instead of mutable `latest`/major convenience tags:

- `minio/minio:RELEASE.2025-09-07T16-13-09Z`
- `redis:7.4.1-alpine`

`compose-security-lint.ps1` subsequently rejects `:latest` image tags fail-closed across all repository Compose fixtures. This does not replace real registry digest signing/provenance or vulnerability scanning gates; it minimizes surprise image changes across the local development stack.
