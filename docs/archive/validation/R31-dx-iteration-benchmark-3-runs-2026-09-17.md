# R31 — Repeated HydroGo DX iteration baseline

**Date:** 2026-09-17  
**Commit under test:** `cd65f0d1cf4389c316ccf9fda21b3bc2ef0ea6f7`  
**Environment:** Windows amd64, Go 1.27.0

## Run

```powershell
pwsh -NoProfile -File scripts/measure-iteration.ps1 `
  -Runs 3 `
  -OutputPath docs/validation/R31-dx-iteration-benchmark-3-runs-2026-09-17.json
```

The script now repeats the complete scaffold → route contract → resource
generation → compatibility sequence and records each run independently.

## Result

| Run | Acceptance duration |
| ---: | ---: |
| 1 | 1,714 ms |
| 2 | 1,158 ms |
| 3 | 1,191 ms |
| **Median** | **1,191 ms** |

Minimum was 1,158 ms and maximum was 1,714 ms. All 3 runs passed and stayed
below the local 15-minute feedback threshold. The full compile and targeted UI
contract are recorded separately as setup measurements; they are not mixed
into the acceptance median.

## Interpretation and limits

This closes the repeatability requirement for the local HydroGo baseline. It
does not compare Payload, browser editing, manual touches, errors, request
latency, RSS, database pool behavior or a second independent customer
resource. The comparative Payload parity gate therefore remains open.
