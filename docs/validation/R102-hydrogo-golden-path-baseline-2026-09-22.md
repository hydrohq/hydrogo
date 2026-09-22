# R102 — HydroGo golden-path iki resource baseline

**Tarih:** 2026-09-22  
**Durum:** HydroGo baseline başarılı; Payload çalışma zamanı smoke ölçümleri
alındı. Tam parity karşılaştırması henüz tamamlanmadı.

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
Bu tablo tek başına Payload parity iddiası değildir. İlk testimonial koşusundaki
9,137 ms soğuk başlangıç outlier'ı ayrıca izlenmelidir; warm koşular
1,620–1,651 ms aralığındadır.

## Payload smoke kanıtı

Aynı Windows makinesinde PTY üzerinden `create-payload-app@3.90.1 -t blank`
ile SQLite tabanlı geçici bir uygulama kuruldu. `Articles` collection'ı için
bir `Articles.ts` dosyası ve `payload.config.ts` kaydı elle eklendi; bu, gerçek
bir Payload koleksiyonunun HydroGo blueprint'ine kıyasla gereken ek dokunuşları
görünür kılar. `npm run generate:types` ve `npm run build` başarılı oldu.

| Ölçüm | Sonuç |
| --- | ---: |
| Payload / Next | 3.90.1 / 16.3.3 |
| İlk `/admin` yanıtı | HTTP 200, yaklaşık 13.7 sn (cold compile) |
| Warm `/admin` | 61–163 ms |
| Warm `GET /api/articles?limit=10` | HTTP 200, 114–180 ms |
| Warm production build | 16,856 ms |
| Uygulama kaynak dosyası | 17 |
| `node_modules` boyutu | yaklaşık 701.6 MB |

İlk denemede PTY olmadan CLI `uv_tty_init ... EBADF` verdi; PTY ile kurulum
başarıyla tamamlandı. Bu, harness engelinin giderildiğini gösterir. Ölçüm yine
de tam parity değildir: kimlik doğrulamalı admin CRUD/browser senaryosu,
seed'lenmiş veri, p95/p99 yük testi ve eşdeğer özellik kapsamı henüz
karşılaştırılmadı. API'nin ilk isteği erişim kuralı hot-reload edilene kadar
403 döndü; sonrasında public read ile 200 alındı. Bu gözlem, Payload'ın
varsayılan erişim/konfigürasyon yükünü ayrıca değerlendirmeyi gerektirir.

Geçici Payload çalışma alanı ölçümden sonra repository dışındaki Temp altında
tutuldu; release artefact'ı değildir.

Ham çıktılar `.tmp/benchmark-module-starter.json` ve
`.tmp/benchmark-article-collection.json` altında üretildi; `.tmp` release
artefact'ı değildir.
