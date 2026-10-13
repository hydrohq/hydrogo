# R35 — Headless starter contract smoke wired to E2E flow

**Date:** 2026-09-17  
**Status:** Successful (script/CI wiring)

## Scope

`scripts/run-e2e.ps1` now runs `examples/headless-starter/npm test` before starting generated browser journeys. Consequently, the TypeScript client contract smoke test is no longer merely an optional manual check, but a mandatory prerequisite for the existing Windows/CI E2E gate.

## Verification

```powershell
Push-Location examples/headless-starter
npm test
Pop-Location
```

Result: **PASS** — TypeScript build and `PASS: headless client contract smoke`.

For full browser workflows, run `scripts/run-e2e.ps1` with `HYDRO_E2E_ADMIN_PASSWORD`. This phase does not close external service, payment provider, or production proxy gates.
