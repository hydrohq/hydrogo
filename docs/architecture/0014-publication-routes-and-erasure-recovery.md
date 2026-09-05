# ADR-0014: Yayın route'ları ve silme geçmişiyle güvenli kurtarma

- **Durum:** Kabul edildi
- **Tarih:** 2026-09-05
- **Kısmen değiştirdiği ADR:** [ADR-0012](0012-content-media-and-portability.md) slug/301 etkinleşme zamanı; [ADR-0013](0013-identity-and-security-boundaries.md) restore sırasında silme kayıtlarının kaynağı.
- **İlişkili ADR:** [ADR-0011](0011-consistency-upgrade-and-recovery.md)
- **Uygulama durumu:** Tasarım kabul edildi; uygulama ve test kanıtları bekliyor.

## Bağlam

Taslak gövdesi canlı revision'dan ayrılmıştı; slug ve SEO değişikliklerinin ne zaman görünür olduğu açık değildi. Eski yedeğe dönüşten sonra silme kayıtlarını uzlaştırmak isteniyordu; aynı DB yedeği, yedekten sonra gerçekleşen silme işlemlerini içermez.

## Yayın kararı

- Taslak slug, canonical/SEO alanları ve gövde revision'a aittir. Taslak kaydetmek canlı route, redirect, sitemap veya public API projection'ını değiştirmez.
- Publish transaction'ı revision işaretçisini, aktif route'u, gerekiyorsa önceki aktif route'tan 301 kaydını, audit/outbox ve publication generation'larını birlikte günceller. İlk yayında önceki aktif route yoksa redirect oluşmaz.
- Slug normalizasyonu, reserved path, benzersizlik ve redirect döngüsü publish sırasında tekrar kontrol edilir. Taslakta uygun görünen path sonradan alınmışsa publish conflict verir; önceki canlı yayın korunur.
- İlk dilimde taslak path rezervasyonu yoktur. İki taslak aynı path'i isteyebilir; ilk başarılı publish kazanır. Hata kullanıcıya görünürdür.
- Önizleme revizyona bağlı özel URL kullanır; public route'u veya shared cache'i değiştirmez. Zamanlanmış yayın aynı publish politikasını sabit revision ve schedule generation ile çalıştırır.
- Public cache yayılımı ADR-0011'in stale bütçesine tabidir. Redirect ve sitemap invalidation'ı da aynı publication olayına dahildir.

## Silme geçmişi ve restore kararı

Bu protokol sıradan unpublish/soft-delete değil, restore sonrasında da uygulanması gereken kalıcı silme veya anonimleştirme talebi içindir.

- Silme niyeti, üretim DB'sinin restore işleminden bağımsız, sunucu dışında korunan dayanıklı bir günlüğe önce kaydedilir ve kalıcılık onayı alınır. Yalnız aynı DB'deki audit/outbox yeterli değildir.
- Kayıt edition kimliği, benzersiz işlem kimliği, sıralı checkpoint, zaman, işlem türü ve yeniden uygulamaya yeterli sabit hedef kimliklerini taşır. Silinen içerik, form gövdesi veya secret taşımaz; kimlikler de erişimi sınırlı ve şifreli tutulur.
- Sonra DB'de silme/anonimleştirme ve audit transaction'ı uygulanır. Dış günlük ile DB arasında atomik transaction varsayılmaz. Niyet kaydedilip DB işlemi başarısız olursa işlem pending kalır ve idempotent tekrar yürütülür; kullanıcıya tamamlandı denmez. Günlük ulaşılamıyorsa kalıcı silme tamamlanmaz.
- Günlükteki niyet geri alınabilir sıradan bir düzenleme değildir. Yetki ve hedef kontrolü niyet yazılmadan yapılır. Aynı işlem kimliğiyle tekrar içerik, revision, ilişki, medya ve arama temizliğini güvenle sürdürebilmelidir.
- Backup manifest'i günlük kimliğini ve gördüğü checkpoint'i içerir. Restore, yedekten bağımsız güncel günlüğü edinir ve uygulanabilir silme niyetlerini staging hedefe idempotent uygular. Yedek sonrası kayıtlar özellikle kapsanır.
- Günlüğün bütünlüğü, sıra sürekliliği ve güncel checkpoint'i doğrulanmadan public trafik, export, worker veya dış egress açılmaz. Eksik günlükle kurtarma izolasyonda kalır; operatör eksik kaynağı kurtarmalıdır. Eski backup'ın checkpoint'i tek başına güncellik kanıtı değildir.
- Günlük retention'ı geri yüklenmesine izin verilen en eski backup ve onun bütün dış kopyalarıyla koordine edilir. Gerekli silme kaydı düşürülmeden önce ilgili eski backup'lar desteklenen restore kümesinden çıkarılır. Günlük ve anahtar kurtarma tatbikatı pilot kapısıdır.
- Compact için ayrı sürekli servis zorunlu değildir: mevcut güvenli dış yedek hedefi üzerinde sürümlü günlük adapter'ı kullanılabilir. Kesin sağlayıcı, eşzamanlı append/checkpoint ve dayanıklılık protokolü R3 deneyinde seçilir. Bu kanıt olmadan kalıcı silme/restore garantisi sunulmaz.

## Alternatifler

Slug değişikliğini taslak kaydında etkinleştirmek taslak/canlı ayrımını bozar. Her taslağa path rezervasyonu ilk dilimde gereksiz yaşam döngüsü ekler. Silme geçmişini yalnız DB yedeğinde tutmak yedek sonrası talepleri kaybeder. Genel DB+harici günlük atomikliği varsaymak yerine önce dayanıklı niyet ve idempotent tamamlama seçilir.

## Etkiler ve uyumluluk

- SQLite/PostgreSQL: route uniqueness, publish conflict ve içerik/route/audit/outbox atomikliği aynı contract kitini geçer. SQL ve locking adapter içinde kalır; yeni DB de aynı semantiği sağlamalıdır.
- Migration: route/revision ilişkileri ve silme işlem kimliği için gerekli metadata uygulama sırasında dialect başına hazırlanır. Henüz SQL migration yayımlanmadı. Eski okuyucu/yazıcı bu semantiği koruyamıyorsa online release ilan edilmez.
- Compact: route cache sınırlı kalır; dış günlük çağrıları deadline/concurrency bütçesi taşır. Sınırsız retry veya bellek kuyruğu yoktur.
- Standard/Distributed: eşzamanlı niyet kayıtları kaybolmamalı; global sıra/checkpoint ve tekrar davranışı provider testinden geçmelidir. Bütün web/worker cache'leri publication generation'ını izler.
- Blue/green: iki slot aynı publish ve silme protokolünü destekler. Restore eski deployment yazmalarını kapatır; session iptali, retained asset ve job karantinası ADR-0011'e tabidir.

## Zorunlu kanıtlar

1. Yayındaki sayfanın taslak slug/SEO değişikliği public URL, canonical ve sitemap'i etkilemez; publish hepsini doğru revision'a taşır.
2. İki içerik aynı path'e publish olurken tek işlem başarılıdır; kaybeden eski canlı durumunu korur. Redirect döngüsü ve publish ortasında hata iki DB'de rollback olur.
3. Pazartesi backup → salı kalıcı silme → çarşamba pazartesiye restore: veri public, export, revision, medya ve aramadan geri görünmez.
4. Niyet onayı sonrası DB crash, günlük kesintisi, eksik/eski checkpoint ve yinelenen silme işlemleri güvenli sonuç verir.
5. Günlük retention/anahtar kaybı durumunda restore trafik açmaz; dış günlük ve backup birlikte temiz ortamda kurtarılır.
