# R55 — S3 endpoint URL boundary

**Date:** 2026-09-17  
**Status:** Successful (local contract)

When provided in URL format, S3-compatible endpoint URLs only accept HTTP/HTTPS schemes. Endpoints containing userinfo, queries, or fragments are rejected fail-closed rather than silently normalized; HTTPS endpoints deterministically enable `UseSSL`.

Verification: `go test ./internal/media` and full pre-push `verify-local` hook. Real provider TLS/IAM/certificate behavior remains an external production gate.
