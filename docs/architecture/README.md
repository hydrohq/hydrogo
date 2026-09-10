# HydroGo Architecture Decision Log

Bu dizin HydroGo'nun uzun ömürlü mimari kararlarının ana kaynağıdır. Uygulama planı iş sırasını, ADR'ler ise sistemin neden bu biçimde kurulduğunu açıklar.

Henüz kabul edilmemiş sorular ve önerilen cevaplar [Architecture Decision Backlog](DECISION_BACKLOG.md) içinde izlenir.

Kodlama riskleri ve açık kanıt kapıları [Uygulama öncesi inceleme](../PRE_IMPLEMENTATION_REVIEW.md), roadmap ve [üretim kapısı kontrol listesinde](../validation/PRODUCTION-GATES.md); güvenlik akışları [Tehdit modeli](../security/THREAT_MODEL.md) içindedir. **Kabul edilmiş ADR tek başına uygulama veya test kanıtı değildir; her ADR'nin gerçek durumu roadmap ve doğrulama raporlarıyla birlikte değerlendirilir.**

Yeni bir geliştirme aşağıdaki konulardan birini değiştiriyorsa ilgili ADR okunmalı ve gerekiyorsa yeni ADR yazılmalıdır:

- Core ile müşteri kodu arasındaki sınır
- Veritabanı, cache, medya veya iş kuyruğu altyapısı
- Dağıtım, migration ve geriye uyumluluk
- Güvenlik sınırı veya varsayılanı
- Modül sözleşmesi ve kod üretimi
- Desteklenen ölçek ve kaynak profilleri

## ADR durumları

- **Önerildi:** Yön belirlenmiştir; uygulama veya benchmark ile doğrulanmalıdır.
- **Kabul edildi:** Yeni geliştirmeler bu karara uymalıdır.
- **Değiştirildi:** Yerine geçen ADR bağlantısı belirtilir.
- **Değiştirildi (kısmen):** Hangi hükümlerin değiştiği ve geçerli kalan kapsam açıkça belirtilir; iki belge birlikte okunur.
- **Reddedildi:** Değerlendirilmiş fakat seçilmemiştir.

Kabul edilmiş bir karar sessizce değiştirilmez. Yeni bilgi kararı geçersiz kılıyorsa eski ADR korunur, yeni ADR yazılır ve iki belge birbirine bağlanır.

## Karar listesi

| ADR | Karar | Durum |
|---|---|---|
| [0001](0001-product-and-deployment-model.md) | Ürün ve müşteri deployment modeli | Kabul edildi |
| [0002](0002-database-portability.md) | SQLite, MySQL ve PostgreSQL desteği | Değiştirildi |
| [0003](0003-cache-architecture.md) | Çok katmanlı profesyonel cache | Kısmen değiştirildi: 0011 |
| [0004](0004-scaling-profiles.md) | Sayısal kaynak profilleri | Değiştirildi |
| [0005](0005-zero-downtime-deployment.md) | Blue/green ve kesintisiz sürüm geçişi | Kısmen değiştirildi: 0011 |
| [0006](0006-extensible-database-adapters.md) | İlk sürüm SQLite/PostgreSQL ve genişletilebilir DB adaptörleri | Kabul edildi |
| [0007](0007-capacity-profiles.md) | Sayısal donanım vaadi olmadan kapasite profilleri | Kabul edildi |
| [0008](0008-module-and-extension-contract.md) | İlk modül/extension sözleşmesi | Değiştirildi: 0010 |
| [0009](0009-site-edition-and-delivery-scope.md) | Site Edition ve aşamalı teslim kapsamı | Kabul edildi; planlandı |
| [0010](0010-sdk-lifecycle-and-edition-manifest.md) | SDK olgunluğu, modül yaşam döngüsü ve manifest | Kabul edildi; planlandı |
| [0011](0011-consistency-upgrade-and-recovery.md) | Cache/işlem tutarlılığı, upgrade ve recovery | Kabul edildi; planlandı |
| [0012](0012-content-media-and-portability.md) | İçerik, medya ve veri taşınabilirliği | Kısmen değiştirildi: 0014; planlandı |
| [0013](0013-identity-and-security-boundaries.md) | Kimlik, yetki ve güvenlik sınırları | Kısmen değiştirildi: 0014; planlandı |
| [0014](0014-publication-routes-and-erasure-recovery.md) | Publish sırasında route/SEO ve restore'dan bağımsız silme geçmişi | Kabul edildi; planlandı |
| [0015](0015-versioned-http-api.md) | Sürümlü HTTP API yolları ve geriye dönük alias | Kabul edildi |
| [0016](0016-structured-page-blocks-and-theme-slots.md) | Yapılandırılmış sayfa blokları ve tema slotları | Kabul edildi |
| [0017](0017-commerce-consistency-and-payments.md) | Commerce tutarlılığı ve ödeme sağlayıcıları | Kabul edildi |
| [0018](0018-filesystem-theme-bundles.md) | Dosya tabanlı tema paketleri ve güvenli slot renderer | Kabul edildi |

Uygulama sırası ve çıkış kapıları [Roadmap](../ROADMAP.md) içinde izlenir.

## Her geliştirmede uygulanacak kontrol

1. Değişikliğin etkilediği ADR'leri belirle.
2. Müşteri edition'ı ile Core sözleşmesini bozmadığını kontrol et.
3. SQLite/PostgreSQL davranışını ve gelecekteki adaptörlere etkisini incele.
4. Cache invalidation ve eski veri riskini değerlendir.
5. Compact profilde gereksiz servis veya sınırsız kaynak tüketimi oluşturmadığını doğrula.
6. Blue/green sırasında eski ve yeni sürümün birlikte çalışabildiğini kontrol et.
7. Migration bakım penceresi gerektiriyorsa bunu release metadata'sında açıkla.
