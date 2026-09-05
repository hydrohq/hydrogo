# ADR-0003: Çok katmanlı profesyonel cache

- **Durum:** Değiştirildi (kısmen)
- **Tarih:** 2026-09-05
- **Yerine geçen hükümler:** [ADR-0011](0011-consistency-upgrade-and-recovery.md)

> L0–L3 katmanları ve profil seçimi geçerlidir. Aşağıdaki genel kesintisizlik ve worker-L1 invalidation ifadeleri tarihsel kalmıştır; çok prosesli fan-out, generation, stale bütçesi ve bounded fallback için ADR-0011 uygulanır.

## Bağlam

HydroGo küçük tek sunucuda ek servis olmadan ve çok instance'lı kurulumlarda verimli çalışmalıdır. HTTP cache eski bir teknoloji değildir; güncel standart RFC 9111 ile tanımlanan, tarayıcı ve edge katmanında isteği uygulamaya ulaşmadan karşılayan en ucuz cache katmanıdır. Dağıtık cache ise farklı bir problemi, instance'lar arasında paylaşım ve koordinasyonu çözer.

## Karar

Cache dört seviyeli tasarlanır:

```text
L0  Request-scope memoization
L1  HydroGo bounded memory cache
L2  Redis/Valkey uyumlu distributed cache (isteğe bağlı)
L3  HTTP reverse proxy / CDN / browser cache
```

| Profil | L1 Memory | L2 Distributed | L3 HTTP/CDN |
|---|---:|---:|---:|
| Compact | Evet | Hayır | Evet |
| Standard | Evet | İsteğe bağlı | Evet |
| Distributed | Evet | Zorunlu | Zorunlu |

Valkey, BSD lisanslı ve Linux Foundation destekli olduğu için self-hosted varsayılan dağıtık cache adayıdır. Redis protokolüyle uyumlu adaptör sağlanır. Redis 8'in lisans seçenekleri farklı yükümlülükler taşıdığından ürün dağıtımında sürüm ve lisans ayrıca değerlendirilir.

## Cache sözleşmesi

Cache yalnız yeniden üretilebilir veri içerir. Cache kaybı veri kaybına veya uygulama kesintisine dönüşmez.

```go
type Cache interface {
    Get(ctx context.Context, key string) ([]byte, bool, error)
    Set(ctx context.Context, key string, value []byte, ttl time.Duration, tags ...string) error
    Delete(ctx context.Context, keys ...string) error
    InvalidateTags(ctx context.Context, tags ...string) error
}
```

Planlanan sürücüler:

- `NoopCache`
- `MemoryCache`
- `ValkeyCache`
- `RedisCache`

Redis ve Valkey aynı protokolü paylaşsa da davranış, sürüm ve lisans farkları nedeniyle ürün konfigürasyonunda adları açıkça belirtilir.

## Cache edilebilen veriler

- Yayımlanmış public HTML
- Public API cevapları
- Menü ve navigasyon ağacı
- Site ayarlarının public alt kümesi
- SEO, sitemap ve RSS çıktıları
- Hesaplanması pahalı, tekrar üretilebilir sorgu sonuçları

## Cache edilmeyen veriler

- Parola, token ve hassas session içeriği
- Audit olayları
- Job'un tek doğruluk kaydı
- Taslak ve özel önizleme
- Para, stok ve ödeme defterinin asıl kaydı
- Yetki kararının uzun ömürlü sonucu

## Profesyonel cache davranışı

- Cache-aside temel okuma kalıbıdır.
- Her anahtar ürün, müşteri, locale, içerik türü, sürüm ve serialization versiyonuyla namespace edilir.
- TTL zorunludur ve aynı anda toplu sona ermeyi önlemek için jitter eklenir.
- Cache stampede, tek instance'ta `singleflight`; dağıtık profilde kısa lease/lock ile sınırlandırılır.
- Negatif cache yalnız güvenli `not found` sonuçlarında ve kısa TTL ile kullanılır.
- Büyük girdiler `max_entry_size` ile reddedilir.
- L1 bellek byte bütçeli ve bounded olur; sınırsız map kullanılmaz.
- Dağıtık cache kesilirse uygulama kontrollü biçimde DB'ye döner ve circuit breaker/backoff uygular.
- Cache hatası içerik yazma transaction'ını geri almaz.
- Hit/miss, latency, eviction, entry bytes, stampede suppression ve invalidation failure ölçülür.

## Invalidation

Anahtar silmeye ek olarak sürümlü namespace kullanılır:

```text
page:{site}:{locale}:{slug}:{published_revision}
menu:{site}:{locale}:{menu_version}
collection:{site}:{locale}:{type}:{publication_version}:{query_hash}
```

Yayınlama transaction'ı yeni revision/version ile birlikte bir outbox olayı yazar. Commit sonrasında worker:

1. L1 cache'i temizler.
2. L2 tag/namespace invalidation gönderir.
3. CDN purge veya surrogate-key invalidation yapar.

Başarısız invalidation yeniden denenir. TTL ve sürümlü anahtarlar kalıcı stale veri riskini sınırlar.

## HTTP cache

- Yalnız public `GET/HEAD` cevapları ortak cache'e uygundur.
- Admin, preview ve yetkili cevaplar `private, no-store` olur.
- ETag ve koşullu istekler desteklenir.
- HTMX fragment ve tam sayfa aynı URL'den dönüyorsa `Vary: HX-Request` kullanılır.
- Hash'li asset'ler `immutable` olarak cache edilir.
- Eski HTML'nin referans verdiği asset'ler cache ömründen daha uzun saklanır.

HTTP cache ürün içinde tek başına “cache sistemi” diye pazarlanmaz; edge, memory ve dağıtık cache ile aynı gözlemlenebilir cache mimarisinin parçasıdır.

## Kaynaklar

- [RFC 9111 — HTTP Caching](https://www.rfc-editor.org/rfc/rfc9111.html)
- [Valkey overview](https://valkey.io/)
- [Redis licenses](https://redis.io/legal/licenses/)
