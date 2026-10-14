# R68 BFF server-side request policy

**Date:** 2026-09-19  
**Status:** Local policy contract verified; deployment session exchange and
browser E2E remain open.

`internal/security/bffpolicy` is the reference boundary for a separately
hosted web frontend. Tests prove that browser bearer credentials and cookies
are rejected/ignored, only `/api/v1` paths cross the boundary, origin and
forwarding headers are stripped, and a server-side Core cookie is required.

```text
go test ./internal/security/bffpolicy
go test ./internal/security ./modules/... ./tests/contract
```

This is deliberately not a production proxy or a claim that BFF session
exchange is complete. A deploying frontend must implement the server-side
session adapter, CSRF boundary and reverse-proxy TLS/drain/rollback drill.
