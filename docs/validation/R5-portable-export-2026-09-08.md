# R5 taşınabilir export/import doğrulaması

**Tarih:** 2026-09-08  
**Roadmap:** R5 / S09  
**Durum:** Allowlist tabanlı SQLite round-trip başarılı; PostgreSQL round-trip CI contract'ı ve büyük SQLite fixture'ı eklendi. Tekrarlanabilir benchmark komutu eklendi; müşteri donanımıyla karşılaştırmalı benchmark hâlâ açık.

`internal/portable` deterministic JSON formatında pages, page revisions, media metadata, Site Edition, form inbox ve News kayıtlarını taşır. Kimlik, revision, locale ve ilişkiler korunur; `users`, `sessions`, MFA/recovery/invite token'ları, audit, outbox ve jobs tabloları formatın allowlist'ine dahil değildir.

Çalıştırılan kontrol:

```text
go test ./internal/portable
```

Contract test source SQLite'dan export alıp ayrı SQLite hedefe import eder; page ve revision korunur, auth state'in dışarı çıkmadığı doğrulanır. 256 page/revision/media kaydından oluşan fixture kimlik, locale, revision ve ilişkileri round-trip sonrasında doğrular. `HYDRO_TEST_POSTGRES_URL` tanımlı CI koşulunda aynı API geçici PostgreSQL schema'larında source/target round-trip yapar; yerel ortamda test skip edilir.

`BenchmarkSQLitePortableRoundTrip256` Windows amd64 geliştirme makinesinde (Intel(R) Core(TM) 7 240H) hedef şema benchmark başlamadan hazırlanarak `-benchtime=10x` son çalıştırmada 15.4 ms/round-trip ölçtü. Aynı makinede çalıştırmalar arasında belirgin oynaklık görüldü (15.4–369.9 ms/op); bu nedenle sonuç deterministik performans veya SLA kanıtı değildir. Ölçüm export + JSON encode/decode + mevcut hedefe transaction içi import maliyetini kapsar; SQLite açılışı ve migration kurulumu kapsam dışıdır. Pilot müşteri donanımında yük altında tekrar ölçülüp RPO/RTO ve teslim süresine bağlanmalıdır.
