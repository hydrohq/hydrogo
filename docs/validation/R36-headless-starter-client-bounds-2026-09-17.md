# R36 — Headless starter client boundary smoke coverage

**Date:** 2026-09-17  
**Status:** Successful (local Node runtime)

`client-contract.mjs` verifies that client timeout settings are validated fail-closed within the `1..120000` ms range. This ensures misconfigured frontend starter settings surface visibly prior to sending requests.

```powershell
Push-Location examples/headless-starter
npm test
Pop-Location
```

Result: **PASS** — build, idempotency, typed error/request-id, bounded response, and timeout boundary checks.
