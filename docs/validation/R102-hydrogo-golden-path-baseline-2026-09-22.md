# R102 — HydroGo golden-path iki resource baseline

**Tarih:** 2026-09-22  
**Durum:** HydroGo tarafı başarılı; Payload karşılaştırması henüz yapılmadı.

## Senaryolar

`scripts/measure-iteration.ps1 -Runs 3` aynı Windows/Go ortamında şu iki
bağımsız kaynak tanımıyla çalıştırıldı:

| Resource | Fixture | Median | Min–max | Kabul |
| --- | --- | ---: | ---: | --- |
| Testimonial | `examples/module-starter/collection.yaml` | 1,651 ms | 1,620–9,137 ms | PASS |
| Article | `examples/benchmark/article-collection.yaml` | 1,546 ms | 1,498–1,551 ms | PASS |

Ölçülen akış her senaryoda proje scaffold, route manifest kontrolü, resource
generation ve generated compatibility check adımlarını içerir. Üç koşunun
her biri 15 dakikalık yerel acceptance eşiğinin altında kaldı.

## Yorum

Bu rapor iki HydroGo resource senaryosunun tekrarlanabilir DX baseline'ıdır.
Payload parity iddiası değildir: aynı makinede Payload kurulumu, manuel dokunuş
sayısı, ilk çalışan admin/API smoke, browser düzenleme, RSS ve p95/p99
karşılaştırması henüz çalıştırılmadı. İlk testimonial koşusundaki 9,137 ms
soğuk başlangıç outlier'ı ayrıca izlenmelidir; warm koşular 1,620–1,651 ms
aralığındadır.

Ham çıktılar `.tmp/benchmark-module-starter.json` ve
`.tmp/benchmark-article-collection.json` altında üretildi; `.tmp` release
artefact'ı değildir.
