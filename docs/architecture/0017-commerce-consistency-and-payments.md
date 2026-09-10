# ADR-0017: Commerce tutarlılığı ve ödeme sağlayıcıları

- **Durum:** Kabul edildi
- **Tarih:** 2026-09-10
- **İlişkili kararlar:** [ADR-0006](0006-extensible-database-adapters.md), [ADR-0011](0011-consistency-upgrade-and-recovery.md), [ADR-0015](0015-versioned-http-api.md)
- **Uygulama durumu:** Commerce domain/service, SQL store, HTTP checkout, provider-neutral payment contract, webhook replay migration'ı ve Stripe/iyzico/PayTR imza doğrulayıcıları uygulandı. iyzico/PayTR ödeme oluşturma ve iade akışları sağlayıcı hesap/API konfigürasyonu ile ayrı sandbox kapısına tabidir.

## Karar

Commerce ayrı ve isteğe bağlı bir modüldür. Etkin değilken route, migration, worker, asset veya ödeme sağlayıcısı yüklemez.

Sipariş oluşturma şu işlemleri tek DB transaction'ında yapar:

1. Idempotency key ile daha önce oluşturulmuş siparişi bulur.
2. Ürün stoklarını atomik olarak rezerve eder.
3. Fiyat ve para birimini ürün snapshot'ı olarak sipariş satırına yazar.
4. Siparişi oluşturur.
5. `order.created` outbox olayını ekler.

İdempotency key aynı kullanıcıda farklı sipariş gövdesiyle tekrar kullanılırsa hata döner. Sipariş transaction'ı ödeme sağlayıcısına ağ çağrısı yapmaz.

Para değerleri minor-unit integer ve ISO currency koduyla tutulur. Stok negatif olamaz. Domain ve application service SQL dialect bilmez; SQLite/PostgreSQL atomik update ve locking davranışını store adapter'ı sağlar.

## Ödeme

Ödeme sağlayıcısı adapter sözleşmesi `CreatePaymentIntent`, imza doğrulamalı webhook ve `Refund` işlemlerini kapsar. Webhook event ID'si ve provider idempotency key'i kalıcı olarak kaydedilir. Bilinmeyen veya imzası geçersiz event reddedilir; ödeme durumu yalnız doğrulanmış provider event'iyle ilerler.

İlk commerce teslimi ödeme sağlayıcısı olmadan test edilebilir. Stripe, iyzico ve PayTR ayrı adapter ve sandbox contract testleriyle eklenir; sağlayıcı SDK ayrıntıları domain'e sızmaz.

HTTP sözleşmesi [commerce-v1.yaml](../api/commerce-v1.yaml) içinde sürümlenir. `/api` uyumluluk alias'ı router tarafından sağlanır; yeni istemciler `/api/v1` kullanır.

Sepet, kullanıcı başına tek aktif kayıt ve ürün başına tek satır kullanır. Sepete ekleme transaction içinde ürün fiyatını snapshot'lar; checkout sırasında güncel ürün/fiyat ve stok yeniden doğrulanır.

Checkout tercihen `cart_id` ile yapılır. Eski doğrudan `items` biçimi geçiş dönemi uyumluluğu için desteklenir; iki biçim aynı idempotency ve stok transaction'ını kullanır.
