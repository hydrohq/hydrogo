# R34 — Headless starter client contract smoke

**Date:** 2026-09-17  
**Status:** Successful (local Node runtime)

## Scope

`examples/headless-starter` now includes a contract smoke test running with dependency-free Node `assert`. The test verifies the following behaviors against the actual compiled client:

- invalid and valid idempotency key boundaries;
- typed `HydroGoError`, safe message, error code, and request ID propagation;
- fail-closed rejection of streaming responses exceeding 2 MiB without `Content-Length`;
- successful resolution of standard typed news responses.

## Verification

```powershell
npm test
```

Result: **PASS** — TypeScript build and `PASS: headless client contract smoke`.

The test does not bind to external APIs, payment providers, or CORS policies; therefore, it does not substitute for real browser/production acceptance gates.
