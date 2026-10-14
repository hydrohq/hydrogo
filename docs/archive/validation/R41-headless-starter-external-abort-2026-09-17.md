# R41 — Headless starter external abort signal

**Date:** 2026-09-17  
**Status:** Successful (local Node runtime)

The contract smoke test verifies that an `AbortSignal` cancellation supplied by the host application is propagated to the client's internal controller and terminates the call with the caller's reason. Consequently, requests do not hang open on route changes, user cancellations, or framework lifecycle teardown.

```powershell
Push-Location examples/headless-starter
npm test
Pop-Location
```

Result: **PASS** — external abort propagation, timeout abort, idempotency, typed error/request-id, and bounded response checks.
