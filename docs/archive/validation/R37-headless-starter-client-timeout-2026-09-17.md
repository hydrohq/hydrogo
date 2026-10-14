# R37 — Headless starter actual timeout behavior

**Date:** 2026-09-17  
**Status:** Successful (local Node runtime)

The contract smoke test now verifies not only timeout configuration boundaries, but also that the `AbortController` signal actually fires and aborts the call when a request fails to complete. This serves as a regression gate preventing stalled requests in browser apps from consuming unbounded resources.

```powershell
Push-Location examples/headless-starter
npm test
Pop-Location
```

Result: **PASS** — timeout abort, idempotency, typed error/request-id, and bounded response checks.
