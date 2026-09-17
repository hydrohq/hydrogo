# Payload parity benchmark protokolü

Bu protokol, HydroGo'nun Payload CMS'e göre geliştirme hızını ölçmek içindir.
Amaç pazarlama iddiası üretmek değil, aynı donanım ve aynı senaryoda tekrar
edilebilir kanıt toplamaktır.

## Sabit senaryo

İki sistemde de `articles` kaynağı oluşturulur:

- `title` (zorunlu, çok dilli)
- `slug` (unique index)
- `body` (rich text)
- `cover` (media)
- `status` (draft/published)
- `publishedAt` (schedule)
- `category` (relation)

Senaryo; migration, seed, admin listeleme/düzenleme, yayınlama, public detail
API'si, validation ve bir browser smoke testi içerir. İş kuralı elle eklenirse
ayrıca kaydedilir.

## Ölçülecek metrikler

Her koşuda aşağıdaki değerleri kaydet:

1. İlk komuttan çalışan admin/API smoke testine kadar geçen süre.
2. Üretilen dosya ve elle değiştirilen dosya sayısı.
3. Başarısız komut/test ve düzeltme sayısı.
4. İlk başarılı çalıştırmada kullanılan komut sayısı.
5. Cold/warm request p50, p95 ve hata oranı.
6. Peak RSS, CPU, veritabanı bağlantısı ve cache durumu.

## Deney koşulları

- Aynı Windows/WSL makinesi, Go/Node/Payload sürümleri ve temiz checkout.
- Aynı SQLite veri seti; ikinci koşuda desteklenen PostgreSQL sürümü.
- Cold cache ve warm cache ayrı raporlanır.
- Ağır dış servisler (ödeme, S3, CDN) mock değilse sağlayıcı ve bölge yazılır.
- Her sistem en az üç kez çalıştırılır; ortanca ve aralık raporlanır.

## HydroGo başlangıç komutları

```bash
hydro project new benchmark-news --preset=news
cd benchmark-news
hydro project check
hydro resource create --name articles --field title:text --field slug:text
hydro generate
hydro migrate
hydro test --package ./...
```

Eksik otomasyon veya manuel wiring adımı gizlenmez; komut çıktısı ve süreleri
`docs/validation/` altında tarihli rapora eklenir.

Tekrarlanabilir yerel DX baseline için `scripts/measure-iteration.ps1` üç koşu
yapar ve `-Runs N` ile 1–10 arasında sınırlandırılmış tekrar sayısını kabul
eder. Çıktı her koşunun süresini, median/min/max değerlerini ve başarısını ayrı
alanlarda taşır. Bu yardımcı ölçüm, karşı sistemin aynı senaryoda çalıştırıldığı
Payload karşılaştırmasının yerine geçmez.

## Kabul kriteri

HydroGo'nun “Payload kadar hızlı” olduğu ancak iki bağımsız resource senaryosunda
ilk çalışan admin/API akışı 30 dakika hedefi içinde ve manuel adım/hata sayıları
ölçülerek doğrulanırsa söylenebilir. Yerel HydroGo iteration script'inin 15 dakikalık
eşiği yalnızca hızlı geri bildirim kapısıdır; 30 dakika parity hedefinin yerine
geçmez. Bu protokol tek başına sonucu kanıtlamaz;
sonuç raporu olmadan parity durumu **açık** kalır.
