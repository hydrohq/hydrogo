# ADR-0004: 512 MB–128 GB kaynak profilleri

- **Durum:** Değiştirildi — [ADR-0007](0007-capacity-profiles.md)
- **Tarih:** 2026-09-05

## Bağlam

HydroGo düşük maliyetli bir VPS'te çalışabilmeli ve daha büyük donanımda ek kaynakları anlamlı biçimde kullanabilmelidir. Bu hedef tek bir otomatik konfigürasyonla değil, açık kaynak profilleri ve ölçülmüş sınırlarla sağlanır.

## Karar

HydroGo dört kaynak profili sunar. Değerler başlangıç bütçesidir; gerçek varsayılanlar benchmark ile doğrulanır.

| Profil | RAM | Uygulama | DB | Cache | Medya |
|---|---:|---|---|---|---|
| Nano | 512 MB–1 GB | Tek proses/instance | SQLite | 16–64 MB L1 | Yerel immutable disk |
| Standard | 2–8 GB | Tek instance, ayrı web/worker olabilir | SQLite, MySQL veya PostgreSQL | 64–512 MB L1, L2 opsiyonel | Yerel veya object storage |
| Performance | 16–32 GB | Blue/green, birden fazla web instance | PostgreSQL/MySQL | L1 + Valkey/Redis | Object storage + CDN |
| Scale-out | 64–128 GB+ toplam kapasite | Çoklu instance ve worker | PostgreSQL/MySQL HA | Dağıtık cache zorunlu | Object storage + CDN |

128 GB desteği tek prosesin bütün belleği kullanması anlamına gelmez. Yük; web instance'ları, worker'lar, dağıtık cache, veritabanı ve medya işleme arasında bölüştürülür.

## Kaynak bütçeleri

Her edition aşağıdaki bütçeleri yayınlar:

- Idle RSS
- Tanımlı trafik altında p50/p95/p99 gecikme
- Maksimum L1 cache byte'ı
- DB connection sınırı
- Worker concurrency
- İstek gövdesi ve upload sınırı
- Aynı anda yapılan görsel dönüşümü
- Graceful shutdown süresi

Nano profilde Core + Site Edition'ın hedef idle RSS değeri 128 MB altında tutulur; kalan bellek işletim sistemi, SQLite page cache, istekler ve güvenli tepe payı içindir. Bu bir release gate olur ve gerçek binary üzerinde ölçülür.

## Konfigürasyon

Profil açıkça seçilir:

```yaml
runtime:
  profile: nano
```

`hydro doctor` makinenin görünen fiziksel/cgroup limitini okuyarak seçilen profil uygun değilse uyarır. Uygulama gizlice veritabanı türü veya topolojisi değiştirmez.

Profil şu varsayılanları belirler:

- `GOMEMLIMIT` için güvenli üst sınır
- L1 cache bütçesi
- DB pool büyüklüğü
- Worker ve medya concurrency
- Maksimum upload ve response buffer'ı
- Timeout ve queue backpressure eşikleri

Kullanıcı bu değerleri değiştirebilir; `hydro doctor` tehlikeli toplam bütçeyi raporlar.

## Dikey ve yatay ölçekleme

- SQLite profili önce dikey ölçeklenir ve public trafik CDN ile origin'den uzaklaştırılır.
- Aynı müşteriye ikinci aktif web instance gerektiğinde PostgreSQL/MySQL + distributed cache profiline geçilir.
- Medya yerel diskten object storage'a taşınmadan çoklu instance açılmaz.
- Worker sayısı idempotent job ve lease sözleşmesi doğrulanmadan artırılmaz.

## Backpressure

Büyük makine desteği sınırsız concurrency anlamına gelmez. Her pahalı kaynak bounded olur:

- HTTP in-flight istekleri
- DB bağlantıları
- Görsel decode/encode işleri
- E-posta/webhook worker'ları
- Export/import işleri
- Cache entry ve toplam byte

Limit aşıldığında sistem bellek tüketmeye devam etmek yerine kuyruklar, kontrollü `429/503` döner veya işi kalıcı kuyruğa alır.

## Doğrulama

Her release en az iki uçta test edilir:

1. 512 MB bellek limiti altında Nano soak testi.
2. Yüksek çekirdek/RAM ortamında concurrency ve throughput testi.

Performans iddiaları istek türü, veri boyutu, cache durumu ve donanım belirtilmeden yayımlanmaz.
