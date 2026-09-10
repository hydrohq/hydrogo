# ADR-0021: Headless browser session ve cross-origin sınırı

- **Durum:** Önerildi
- **Tarih:** 2026-09-10
- **İlişkili kararlar:** [ADR-0013](0013-identity-and-security-boundaries.md), [ADR-0015](0015-versioned-http-api.md), [ADR-0020](0020-headless-first-admin-studio-and-frontend-templates.md)

## Bağlam

Headless frontend ayrı deployment olabilir. Mevcut Core oturumu host-only, `HttpOnly`, `SameSite=Lax` cookie ve çift-gönderimli CSRF değeri kullanır. Bu değerler aynı origin integrated UI için uygundur; rastgele bir üçüncü taraf frontend origin'ine credentialed CORS açmak, hem cookie hem CSRF hem de canonical host sınırını genişletir.

Bir starter'ın `fetch(..., { credentials: "include" })` kullanabilmesi tek başına güvenli cross-origin oturum sözleşmesi değildir. CORS allowlist, preflight, `Origin` doğrulaması, cookie domaini, CSRF token bootstrap'i, preview cache bypass'ı ve redirect sonrası checkout dönüşümleri birlikte tasarlanmalıdır.

## Önerilen karar

İlk headless teslimde iki ayrı erişim modu açıkça ayrılır:

1. **Public read:** Published `/api/v1` projection'ları credential gerektirmez. CORS, varsayılan olarak kapalıdır; açılırsa yalnız yapılandırılmış origin allowlist'i için `GET`, `HEAD` ve açık response header'larıyla çalışır. `*` ile credential birleştirilmez.
2. **Browser mutation:** Cart, checkout, account ve preview gibi session isteyen browser işlemleri aynı-site deployment veya frontend BFF üzerinden Core'a gider. Ayrı bir browser origin'i Core oturum cookie'sine doğrudan erişmez ve Core başlangıçta credentialed CORS açmaz.

Admin Studio her zaman Core origin'inde kalır. OIDC, public API key, personal access token veya mobile token bu ADR'nin kapsamı dışındadır; bunlar ayrı authorization ve veri yaşam döngüsü kararı ister.

## BFF sözleşmesi

Bağımsız storefront, kullanıcı tarayıcısına Core session cookie'sini yaymak yerine kendi domainindeki BFF'yi kullanabilir:

- Tarayıcı yalnız storefront/BFF session'ını görür.
- BFF, Core'a server-to-server çağrıda tenant/deployment kimliği ve kullanıcı oturumu için açıkça tasarlanmış adapter kullanır; mevcut browser cookie'sini gizlice proxy etmez.
- CSRF BFF'nin browser sınırında uygulanır; Core mutation policy, idempotency, audit ve payment redirect doğrulamasını yine kendi katmanında uygular.
- Preview token'ı URL query'sinde kalıcı cache key olarak kullanılmaz; kısa ömürlü, audience-bound, `Cache-Control: private, no-store` işlemidir.

Bu belge BFF token exchange implementasyonu tanımlamaz. İlk deney yalnız same-site reverse proxy topolojisi veya server-rendered BFF fixture'ı ile kanıtlanabilir.

## Güvenlik ve operasyon etkileri

- `Origin` allowlist deploy config'de explicit olur; wildcard, regex veya reflected origin kabul edilmez.
- CORS preflight yanıtları cache'te origin-aware `Vary: Origin, Access-Control-Request-Method, Access-Control-Request-Headers` taşır.
- Public CORS header'ları admin, auth, media-private, webhook, health/metrics veya mutation endpoint'lerine uygulanmaz.
- `compact` profil yeni worker, cache veya browser credential servisi yüklemez.
- Blue/green'de allowlist değişikliği eski/yeni deployment arasında güvenli örtüşme ve rollback planı ister.

## Kabul kapıları

Bu ADR ancak aşağıdakiler kanıtlandıktan sonra kabul edilir:

1. Unknown origin public API response'unda CORS header alamaz; allowlisted origin yalnız published `GET/HEAD` çağrısında doğru `Vary` header'ını alır.
2. Cross-origin `POST` / credentialed request preflight ve gerçek istekte reddedilir.
3. Ayrı repository starter, same-site veya BFF fixture üzerinden login → cart → checkout redirect happy path'ini e2e geçer.
4. Preview token, private medya veya draft içerik cross-origin public cache'e sızmaz.
5. SQLite/PostgreSQL ve compact profil contract'ları ek ağ veya sınırsız kaynak davranışı oluşturmadan geçer.

## Alternatifler

- **Credentialed CORS doğrudan Core'a:** kısa vadede frontend kodunu azaltır, fakat cookie/CSRF ve origin yönetimi için yeni saldırı yüzeyi açar.
- **Her client için bearer token:** mobile/API kullanımını basitleştirebilir; token verme, scope, revocation, sızıntı ve audit yüzeyi yeni bir ürün alanıdır.
- **Yalnız integrated frontend:** mevcut güvenlik varsayımlarını korur ancak ADR-0020'nin bağımsız frontend hedefini karşılamaz.
