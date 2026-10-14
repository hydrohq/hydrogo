# R42 — Headless starter origin validation

**Date:** 2026-09-17  
**Status:** Successful (local Node runtime)

The headless client constructor now requires an absolute HTTP(S) origin and rejects embedded credentials along with `javascript:` and other schemes fail-closed. This ensures misconfigured or copy-pasted URLs produce a visible `RangeError` before reaching the fetch layer.

```powershell
Push-Location examples/headless-starter
npm test
Pop-Location
```

Result: **PASS** — origin scheme/credential boundaries, external abort, timeout, idempotency, typed error/request-id, and bounded response checks.
