# R107 — Compact/Standard/Distributed capacity baseline

**Tarih:** 2026-09-22  
**Commit:** `1e8867f31df42397b038c794a683433522fed58f`  
**Durum:** PASS (yerel Windows smoke)

`scripts/measure-capacity.ps1 -WarmupRequests 100 -SampleCount 20` ile her
profilde `/healthz` ve `/readyz` warm-up sonrası process kaynakları örneklendi.

| Profil | Peak working set | Peak private memory | Handles |
| --- | ---: | ---: | ---: |
| compact | 92.2 MiB | 121.9 MiB | 195 |
| standard | 94.8 MiB | 124.6 MiB | 212 |
| distributed | 101.5 MiB | 132.6 MiB | 302 |

Bu ölçüm Core profile farkını görünür kılar; SQLite tek-process smoke'tur ve
customer workload/SLA, harici PostgreSQL/Redis/proxy/media/queue kapasitesi
değildir. Ham JSON `.tmp/capacity-20260922.json` altında tutulur.
