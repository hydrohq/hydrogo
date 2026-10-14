# R19 local capacity profile baseline — 2026-09-17

`scripts/measure-capacity.ps1` now provides a repeatable smoke harness for the
three capacity profiles. It builds one binary, migrates isolated SQLite files,
starts one process per profile, waits for health/readiness, warms `/healthz`,
and samples process memory/handles/CPU before clean shutdown.

## Run

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\measure-capacity.ps1 `
  -OutputPath .\docs\validation\R19-local-capacity-profile-baseline-2026-09-17.json `
  -WarmupRequests 5 -SampleCount 3 -SampleIntervalMs 100
```

## Observed baseline

Windows amd64, Go 1.27.0, revision `1707f25`:

| Profile | Peak working set | Peak private memory | Handles |
| --- | ---: | ---: | ---: |
| Compact | 91.6 MiB | 118.5 MiB | 188 |
| Standard | 90.2 MiB | 121.7 MiB | 221 |
| Distributed | 94.0 MiB | 127.0 MiB | 298 |

All three profiles returned health/readiness successfully and exited cleanly.
The complete samples are in the accompanying JSON file.

## Limitations

This is a process-level local baseline, not customer capacity certification. It
uses SQLite and one process, and does not measure PostgreSQL, Redis/Valkey,
reverse proxy, external media, queue depth, realistic traffic, 24-hour soak,
or SLO limits. Those production gates remain open.
