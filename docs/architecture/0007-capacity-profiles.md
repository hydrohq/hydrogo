# ADR-0007: Sayısal donanım vaadi olmadan kapasite profilleri

- **Durum:** Kabul edildi
- **Tarih:** 2026-09-05
- **Değiştirdiği ADR:** [ADR-0004](0004-scaling-profiles.md)

## Bağlam

Belirli RAM rakamlarını ürün vaadi yapmak yanıltıcıdır. Aynı RAM miktarında CPU, disk, ağ, veri büyüklüğü ve trafik şekli tamamen farklı sonuçlar üretir. HydroGo küçük kurulumlarda sade kalmalı ve ihtiyaç büyüdüğünde aynı iş/modül sözleşmeleriyle farklı topolojiye geçebilmelidir.

## Karar

HydroGo sayısal donanım paketleri yerine üç kapasite/topoloji profili sunar:

| Profil | Topoloji | DB | Cache | Medya |
|---|---|---|---|---|
| Compact | Tek proses veya aynı binary web+worker | SQLite | Bounded memory + HTTP | Yerel immutable disk |
| Standard | Tek veya ayrılmış web/worker | PostgreSQL; uygun durumda SQLite | Memory, dağıtık cache opsiyonel | Yerel veya object storage |
| Distributed | Çoklu web ve worker | PostgreSQL HA/managed | Memory + Valkey/Redis + HTTP/CDN | Object storage + CDN |

Profil donanım sınıfını değil, sistem topolojisini tanımlar.

## Kaynak yönetimi

Her profil ve edition gerçek benchmark sonucuna göre varsayılan sağlar:

- Go memory limit
- L1 cache byte bütçesi
- DB connection pool
- Worker concurrency
- Medya decode/encode concurrency
- HTTP in-flight ve body limitleri
- Queue/backpressure eşikleri
- Graceful shutdown süresi

Kullanıcı açıkça override edebilir. `hydro doctor` görünen fiziksel/cgroup limit, CPU ve disk koşullarına göre toplam bütçeyi denetler ve öneri verir; uygulama gizlice topoloji veya veritabanı değiştirmez.

## Ölçekleme yolu

```text
Compact
  → PostgreSQL'e geçiş
Standard
  → object storage + distributed cache
Distributed
  → web/worker yatay ölçekleme
```

Public trafik önce doğru HTTP cache ve CDN ile origin'den uzaklaştırılır. Veritabanı veya instance sayısı yalnız ölçüm gerektiriyorsa artırılır.

## Ürün iddiası

HydroGo sabit RAM veya eşzamanlı kullanıcı sayısı vaat etmez. Her release/edition için referans iş yüküyle şu değerleri yayımlar:

- Donanım ve topoloji
- Veri seti
- Cache sıcak/soğuk durumu
- Okuma/yazma oranı
- p50/p95/p99 gecikme
- Throughput ve hata oranı
- Peak/idle bellek

## Sonuçlar

- Pazarlama iddiası ölçülebilir kalır.
- Küçük kurulumlar dağıtık altyapı yükü taşımaz.
- Büyük kurulumlar aynı domain ve modül sözleşmeleriyle büyür.
- Destek sınırı donanım etiketiyle değil, yayımlanmış benchmark ve sertifikalı topolojiyle anlaşılır.
