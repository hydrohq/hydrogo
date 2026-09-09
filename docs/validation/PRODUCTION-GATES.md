# Production gate evidence checklist

Bu liste contract testlerin yerine geçmez; gerçek ortamda açık kalan roadmap kapılarının kanıt formatını sabitler.

## Otomatik preflight

Windows geliştirici/CI ortamında aşağıdaki komut tüm otomatik kapıları aynı sırayla çalıştırır:

```powershell
$env:HYDRO_TEST_POSTGRES_URL = "postgres://user:password@127.0.0.1:5432/hydro_contract?sslmode=disable"
powershell -ExecutionPolicy Bypass -File .\scripts\verify-production-gates.ps1
```

Runner; format, vet, tam test, PostgreSQL contract ortamı zorunluluğu, race-sensitive paketler, reproducible `hydro` binary ve portable benchmark kontrollerini çalıştırır. PostgreSQL URL yoksa veya contract testleri skip edilebilecek durumdaysa başarısız olur. Bu preflight gerçek reverse proxy/process restart, temiz ortam restore, production key ceremony, harici yedek/key recovery veya müşteri RPO/RTO ölçümünü tamamlanmış saymaz.

| Gate | Required environment | Evidence command | Pass condition |
| --- | --- | --- | --- |
| S06 proxy/restart/rollback | two deployable binaries, reverse proxy, retained asset root | `hydro upgrade check`, proxy smoke suite, old/new process restart log | green traffic, old writes fenced, rollback serves retained assets |
| S06 long snapshot budget | PostgreSQL 18, representative table and load | `go test ./tests/contract -run TestPostgresConcurrentIndexWaitsForLongSnapshotContract` | lock wait and release duration recorded |
| S07 sold-profile restore | empty SQLite and PostgreSQL targets, media snapshot, key escrow | `hydro doctor`, restore contract suite, marker inspection | sessions revoked, egress quarantined, old writes blocked, media checksum match |
| S07 external backup/key recovery | external object store and separately controlled key escrow | provider restore drill and key rotation log | clean target opens only after controls and key verification |
| R5 portable benchmark | representative customer fixture and supported OS/DB pair | `go test ./internal/portable -run '^$' -bench BenchmarkSQLitePortableRoundTrip256 -benchtime=10x` | result recorded with hardware, DB, fixture and commit |
| R5 release provenance | production signing key ceremony and provenance store | `go test ./internal/release -run TestManifestProvenanceAttestationBindsSubject` plus signed publication log | subject digest, builder and source revision independently verifiable |
| R6 distributed cache | two web/worker instances, shared L2, partition injection | fan-out/reconnect/load drill | bounded fallback, generation reconciliation and no stale refill |

Her gate için sonuç, tarih, commit SHA, profil, OS/DB sürümü ve başarısızlık logları validation raporuna eklenir. Production credentials veya private keys repoya yazılmaz.
