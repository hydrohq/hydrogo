# E0 WSL Redis Distributed Admission Drill — 2026-09-21

`scripts/verify-wsl-redis-drill.ps1` now provides a repeatable operator gate:

- checks that its disposable port is unused;
- starts a non-persistent Redis daemon in WSL;
- runs `CGO_ENABLED=1 go test -race ./internal/security -count=1` with
  `HYDRO_TEST_REDIS_URL` set;
- shuts Redis down through a shell trap.

Observed pass on port `6393`:

```text
ok  github.com/ardaozyildirim/hydrogo/internal/security
PASS: WSL Redis distributed admission/race drill
```

This certifies the local shared-store contract and race behavior. It does not
replace a multi-node failover, network partition, or clock-skew exercise.
