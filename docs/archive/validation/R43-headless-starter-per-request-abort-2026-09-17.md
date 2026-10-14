# R43 — Headless starter per-request abort option

**Date:** 2026-09-17  
**Status:** Successful (local Node runtime)

The headless client's public read and Commerce methods now accept `HydroGoRequestOptions` as the final parameter. Consequently, `signal` can abort specific requests on pages sharing the same client instance; constructor global signal and timeout behaviors remain backward-compatible.

```powershell
Push-Location examples/headless-starter
npm test
Pop-Location
```

Result: **PASS** — per-request abort propagation, global/timeout abort, origin, idempotency, typed error/request-id, and bounded response checks.
