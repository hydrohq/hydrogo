# Production gate evidence checklist

This list does not replace contract tests; it pins the evidence format for roadmap gates remaining open in the real environment.

## Automated preflight

In the Windows developer/CI environment, the following command runs all automated gates in the same order:

```powershell
$env:HYDRO_TEST_POSTGRES_URL = "postgres://user:password@127.0.0.1:5432/hydro_contract?sslmode=disable"
$env:HYDRO_TEST_REDIS_URL = "redis://127.0.0.1:6379/15"
powershell -ExecutionPolicy Bypass -File .\scripts\verify-production-gates.ps1
```

Equivalent preflight in Linux/macOS or Linux CI environment:

```bash
export HYDRO_TEST_POSTGRES_URL='postgres://user:password@127.0.0.1:5432/hydro_contract?sslmode=disable'
export HYDRO_TEST_REDIS_URL='redis://127.0.0.1:6379/15'
bash ./scripts/verify-production-gates.sh
```

Runners execute format, vet, full suite, PostgreSQL contract environment requirement (including the generated-preset module-isolation contract), race-sensitive packages, reproducible `hydro` binary, portable benchmark, and (when configured) the Redis/Valkey atomic limiter contract. The local `scripts/measure-capacity.ps1` harness provides a separate process-level profile baseline (`R19-local-capacity-profile-baseline-2026-09-17`), but is not part of the production gate because it uses SQLite and synthetic traffic. It fails if there is no PostgreSQL URL or if a configured contract fails. Redis is recorded as `PENDING` when `HYDRO_TEST_REDIS_URL` is absent; this preflight does not consider the actual reverse proxy/process restart, clean environment restore, production key ceremony, external backup/key recovery, Redis failover/partition/clock-skew behavior, or customer RPO/RTO measurement as completed.

| Gate | Required environment | Evidence command | Pass condition |
| --- | --- | --- | --- |
| S06 proxy/restart/rollback | two deployable binaries, reverse proxy, retained asset root | `hydro upgrade check`, proxy smoke suite, old/new process restart log | green traffic, old writes fenced, rollback serves retained assets |
| S06 long snapshot budget | PostgreSQL 18, representative table and load | `go test ./tests/contract -run TestPostgresConcurrentIndexWaitsForLongSnapshotContract` | lock wait and release duration recorded |
| S07 sold-profile restore | empty SQLite and PostgreSQL targets, media snapshot, key escrow | `hydro doctor`, restore contract suite, marker inspection | sessions revoked, egress quarantined, old writes blocked, media checksum match |
| S07 external backup/key recovery | external object store and separately controlled key escrow | provider restore drill and key rotation log | clean target opens only after controls and key verification |
| R5 portable benchmark | representative customer fixture and supported OS/DB pair | `go test ./internal/portable -run '^$' -bench BenchmarkSQLitePortableRoundTrip256 -benchtime=10x` | result recorded with hardware, DB, fixture and commit |
| R5 release provenance | production signing key ceremony and provenance store | `go test ./internal/release -run TestManifestProvenanceAttestationBindsSubject` plus signed publication log | subject digest, builder and source revision independently verifiable |
| R6 distributed cache | two web/worker instances, shared L2, partition injection | fan-out/reconnect/load drill | bounded fallback, generation reconciliation and no stale refill |

For each gate, the result, date, commit SHA, profile, OS/DB version, and failure logs are appended to the validation report. Production credentials or private keys are not written to the repository.
