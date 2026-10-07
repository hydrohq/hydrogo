# Blueprint → Admin ilk CRUD ölçümü

**Tarih:** 2026-10-07  
**Durum:** Kısmi; temiz proje scaffold'ından ilk CRUD'a toplam süre ölçülemedi.  
**Kapsam:** `hydro project new`, `hydro new resource`, `hydro project check` ve
önceden composition'ı bulunan Project Edition'da gerçek generated Admin CRUD.

## Temiz Client Edition scaffold ölçümü

Linux çalışma ortamında HydroGo CLI bir kez derlendi; derleme süresi ölçümlere
dahil edilmedi. Her tekrar ayrı, boş bir dizinde `editorial` preset'iyle
çalıştırıldı. Kaynak komutu `title:text:required,body:richtext` Blueprint'ini
oluşturdu. Sonra konfigürasyon ve generated registry
`hydro project check --config config.yaml` ile doğrulandı.

| Tekrar | `project new` | `new resource` | `project check` | Üretilen resource dosyası |
| --- | ---: | ---: | ---: | ---: |
| 1 | 25 ms | 34 ms | 28 ms | 40 |
| 2 | 24 ms | 32 ms | 25 ms | 40 |
| 3 | 24 ms | 34 ms | 27 ms | 40 |
| Medyan | **24 ms** | **34 ms** | **27 ms** | — |

Komutlar sıcak CLI binary ile çalıştı; build/kurulum, dış servis hazırlığı,
migration uygulaması, Go Edition bileşimi ve tarayıcı CRUD süresi dahil değildir.
Ölçüm Payload karşılaştırması değildir ve Windows'taki eski scaffold
ölçümleriyle aynı koşul sayılmaz.

## Gerçek Edition aktivasyon denemesi

Üçüncü kaynak (`milestone2`) geçici bir Project Edition worktree'sinde
oluşturuldu. Generated Admin registry ve `RegisterGeneratedAdmin` kaynağı
Blueprint'ten otomatik güncellendi; Admin menüsüne kaynak ID'si eklenmedi.
Edition'ın var olan `demoPolicy`'si yalnızca loopback test fixture'ında
kullanıldı. Üretim Edition'ında bu politika kullanılmamalı.

Elle yapılan aktivasyon iki Go dosyasında dört düzenlemeydi:

- `custom/composition.go`: generated route paketi importu ve
  `RegisterMilestone2Route` çağrısı.
- `cmd/server/main.go`: generated migration paketi importu ve gözden geçirilmiş
  `MigrationSQL`'ın açık migrate komutuna eklenmesi.

`go test ./examples/project-edition/... -count=1` geçti. Mevcut CRUD E2E
senaryosunun geçici uyarlanmış kopyası Chromium'da generated Admin üzerinden
kayıt oluşturma, API'den okuma, düzenleme ve silmeyi doğruladı: **1/1 geçti**
(13.3 saniye). Bu E2E kopyası deneme içindi; repo'ya yeni bir test dosyası
eklenmedi.

## Açık kabul maddesi

`hydro project new` config/theme/migration dizinleri üretir, çalıştırılabilir Go
Edition modülü veya Admin server composition'ı üretmez. Dolayısıyla bu temiz
scaffold üzerinde tek başına `hydro new resource` sonrası gerçek browser CRUD
başlatılamaz. Geçerli authorization/tenant politikası ve migration onayı da
Edition'a ait kalır. İlk temiz scaffold'tan çalışan müşteri CRUD'una kadar
duvar saati ve gerçek geliştirici müdahale süresi henüz ölçülmüş değildir.
Sonraki geliştirme için hedef, bu güvenlik sınırlarını koruyarak Edition
composition'ı açan tekrarlanabilir, testli bir starter/golden path sağlamaktır.
