# R54 — S3 object-key and public-base boundary

**Date:** 2026-09-17  
**Status:** Successful (local contract)

The S3-compatible media adapter now applies consistent bounded opaque-object-key validation across all `Put`, `Get`, `Delete`, `PublicURL`, and `PresignedURL` operations. Empty segments, `.`/`..`, backslashes, control characters, and overly long keys are rejected fail-closed.

Public/CDN base URLs also cannot contain credentials, queries, or fragments; HTTPS requirements and loopback-only HTTP development exceptions are preserved.

Verification: `go test ./internal/media` and pre-push `scripts/verify-local.ps1`. Real S3/CDN IAM, purge, retention, and failover evidence remains open as a production gate.
