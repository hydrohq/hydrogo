# R65 — Local production drills and capacity baseline (2026-09-17)

## Executed items

```text
pwsh -NoProfile -File scripts/verify-local-production-drills.ps1
pwsh -NoProfile -File scripts/measure-capacity.ps1 -OutputPath .tmp/gates/capacity-local-20260917.json -WarmupRequests 10 -SampleCount 3 -SampleIntervalMs 100
```

## Result

- Isolated SQLite binary/readiness smoke: PASS (`/healthz` and `/readyz` 200).
- Backup, release, and portable contract tests: PASS.
- Blue-green cache rollback stability (20 repetitions): PASS.
- Compact/Standard/Distributed local SQLite process baseline: PASS.
- PostgreSQL contract/restore: PENDING; `HYDRO_TEST_POSTGRES_URL` was not configured in this environment.

## Baseline (Windows/Go 1.27, 10 warmup, 3 samples)

| Profile | Peak working set | Peak private memory | Handles |
|---|---:|---:|---:|
| Compact | 87.95 MiB | 119.35 MiB | 198 |
| Standard | 90.40 MiB | 122.07 MiB | 211 |
| Distributed | 94.65 MiB | 127.47 MiB | 297 |

This measurement is a single-process SQLite smoke test; it does not represent customer SLA, PostgreSQL/Redis/S3/CDN capacity, or Distributed certification. Raw JSON evidence was written to `.tmp/gates/capacity-local-20260917.json` and omitted from Git due to its temporary nature.
