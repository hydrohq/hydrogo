# R53 — Generated client origin-bypass hardening

**Date:** 2026-09-17  
**Status:** Successful (local generator contract)

`HydroClient` path boundary prevents backslash-normalized URL bypasses fail-closed. Because URL parsers may resolve inputs like `"/\\evil.example"` to an external origin, checking only leading slash prefixes was insufficient; the resolved URL origin is now explicitly validated against the configured `baseUrl` origin.

Verification:

- `go test ./internal/projectgen`
- `scripts/verify-local.ps1`

This check mitigates credential exfiltration risk on generated clients; it does not replace server authorization or production proxy policies.
