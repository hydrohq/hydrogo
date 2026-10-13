# R107 — Compact/Standard/Distributed capacity baseline

**Date:** 2026-09-22  
**Commit:** `1e8867f31df42397b038c794a683433522fed58f`  
**Status:** PASS (local Windows smoke)

Sampled process resources following `/healthz` and `/readyz` warm-up across each profile using `scripts/measure-capacity.ps1 -WarmupRequests 100 -SampleCount 20`.

| Profile | Peak working set | Peak private memory | Handles |
| --- | ---: | ---: | ---: |
| compact | 92.2 MiB | 121.9 MiB | 195 |
| standard | 94.8 MiB | 124.6 MiB | 212 |
| distributed | 101.5 MiB | 132.6 MiB | 302 |

This measurement makes Core profile differences visible; it is a single-process SQLite smoke test and does not represent customer workload/SLA or external PostgreSQL/Redis/proxy/media/queue capacity. Raw JSON is preserved under `.tmp/capacity-20260922.json`.
