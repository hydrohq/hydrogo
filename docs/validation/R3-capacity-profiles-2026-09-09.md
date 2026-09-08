# R3 kapasite profili sözleşmesi doğrulaması

**Tarih:** 2026-09-09  
**Roadmap:** R3 S05-C / R5 Standard profil hazırlığı  
**Durum:** Profil ve bounded budget sözleşmesi başarılı; gerçek trafik/benchmark sertifikasyonu açık.

`internal/capacity` artık ADR-0007'deki `compact`, `standard` ve `distributed` profillerini typed bir sözleşme olarak yayımlar. Her profil DB pool, L1 cache, worker/queue, HTTP body, medya ve graceful shutdown bütçesi taşır. Varsayılanlar ölçüm girişidir; donanım veya throughput garantisi değildir.

`Config` profile'ı YAML (`profile`), `HYDRO_CAPACITY_PROFILE` veya `--capacity-profile` ile alır ve bilinmeyen profilleri başlangıçta reddeder. `database.OpenWithOptions` pool sınırlarını profile bütçesine uygular; `hydro serve`, migration ve admin komutları aynı profile budget'ını kullanır. Compact cache/queue/media değerleri eski davranışla aynıdır; Standard ve Distributed değerleri ilk ölçüm varsayımlarıdır.

`hydro doctor` migration uygulamadan aynı profile bütçesiyle DB ping/checksum uyumluluğunu ve medya dizinini kontrol eder; başarılı kontrolde seçilen profile ve sınırları operatöre yazdırır.

Çalıştırılan kontroller:

```text
go test ./internal/capacity ./internal/config ./internal/database ./cmd/hydro
go vet ./internal/capacity ./internal/config ./internal/database ./cmd/hydro
go test ./cmd/hydro -run TestRunMigrateAndCreateAdmin
```

Gerçek Compact blue/green trafik, p95/p99 ve peak memory ölçümü; Standard PostgreSQL connection pool benchmark'ı; Distributed L2/fan-out ve backpressure deneyleri bu doğrulamanın kapsamı dışındadır.

`BenchmarkCacheCoalesces64ConcurrentLookups` Windows amd64 geliştirme makinesinde `-benchtime=10x` ile 202 µs/op ölçtü ve her iterasyonda 64 lookup için loader çağrısını 1'de tuttu. Bu local benchmark müşteri kapasite/SLA kanıtı değildir; gerçek trafik ve profile soak ölçümü ayrıca gerekir.
