# R30 — HydroGo DX iteration baseline

**Date:** 2026-09-17  
**Commit:** `3138122ca239c9b23197ae519c45cfba3fd32562`  
**Environment:** Windows amd64, Go 1.27.0

## Run

```powershell
pwsh -NoProfile -File scripts/measure-iteration.ps1 `
  -OutputPath docs/validation/R30-dx-iteration-benchmark-2026-09-17.json
```

The script ran the targeted UI contract, full compile, first project scaffold,
route contract, first resource generation and generated compatibility check.
All steps exited with code 0.

## Result

| Step | Duration |
| --- | ---: |
| Targeted UI contract | 2,865 ms |
| Full compile | 11,018 ms |
| First project scaffold | 401 ms |
| First route contract | 17 ms |
| First resource generation | 509 ms |
| Generated compatibility check | 472 ms |
| **DX acceptance steps total** | **1,399 ms** |

The measured acceptance steps are under the local 15-minute feedback threshold.
The raw machine-readable result is in
`R30-dx-iteration-benchmark-2026-09-17.json`.

## Interpretation and limits

This proves a repeatable local HydroGo baseline, not Payload parity. The
Payload protocol requires the same article scenario in both systems, at least
three runs, manual-touch/error counts, browser smoke, request latency and
resource measurements. Those comparative and customer-sized measurements
remain open.
