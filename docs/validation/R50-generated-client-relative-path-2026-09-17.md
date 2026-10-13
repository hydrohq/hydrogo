# R50 — Generated client relative-path boundary

**Date:** 2026-09-17  
**Status:** Successful (local generator contract)

Generated `HydroClient` now accepts only relative API paths in `mutation()` and internal request layers. Absolute or protocol-relative URLs raise a fail-closed `RangeError`, preventing configured API keys from being transmitted to unintended hosts.

Verification:

- `go test ./internal/projectgen`
- `scripts/verify-local.ps1`

This check provides a client-side security boundary; it does not replace server-side authorization or production proxy/HTTPS evidence.
