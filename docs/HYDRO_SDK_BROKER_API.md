# Hydro SDK artifact broker API sözleşmesi

Bu belge, public `hydro` CLI ile private runtime artifact servisi arasındaki
minimum sözleşmedir. Broker kaynak kodu veya GitHub PAT'i istemciye vermez;
yalnızca kısa ömürlü, proje/sürüm bağlı bir indirme grant'i üretir.

## Kimlik doğrulama

`hydro login`, OAuth 2.1 Authorization Code + PKCE veya headless cihaz akışı
ile kullanıcı oturumu açar. Refresh token yalnızca işletim sistemi credential
store'unda tutulur. CLI log'larında access/refresh token, Authorization header
ve indirme URL'si görünmez.

Broker istekleri:

```text
POST /v1/runtime/grants
Authorization: Bearer <short-lived access token>
Content-Type: application/json
```

```json
{
  "project_id": "project-123",
  "core_version": "1.4.0",
  "sdk_version": "1.2.0",
  "target": "linux-amd64",
  "edition": "growth",
  "features": ["news", "media"]
}
```

Broker; kullanıcının organization/project entitlement'ını, edition/feature
uyumunu ve target desteğini sunucu tarafında kontrol eder. Repository membership
tek başına lisans veya artifact erişimi sayılmaz.

## Başarılı grant

HTTP `200` cevabı en fazla 64 KiB JSON olmalı ve şu alanları içermelidir:

```json
{
  "access_token": "opaque-short-lived-token",
  "project_id": "project-123",
  "core_version": "1.4.0",
  "sdk_version": "1.2.0",
  "artifact_url": "https://artifacts.example/v1/…",
  "issued_at": "2026-09-16T12:00:00Z",
  "expires_at": "2026-09-16T12:10:00Z"
}
```

Grant en fazla 24 saat geçerli olabilir; normal indirme grant'i birkaç dakika
ile sınırlanmalıdır. URL HTTPS olmalı, credentials/fragment içermemeli ve
redirect takip edilmemelidir. CLI, grant alanlarını lock ve release manifestiyle
eşleştirmeden indirme başlatmaz.

## Hata sözleşmesi

| HTTP | Anlam | CLI davranışı |
| --- | --- | --- |
| 400 | Geçersiz target/sürüm/JSON | Kullanıcıya düzeltilebilir alanları söyle, token gösterme |
| 401 | Oturum geçersiz | Yerel oturumu temizle, yeniden login öner |
| 403 | Entitlement yok | Erişim reddedildi; repository/credential ayrıntısı gösterme |
| 404 | Artifact yok | Sürüm/target desteğini kontrol etmeyi öner |
| 409 | Lock/edition uyumsuz | Lock ve proje ayarlarını karşılaştırmayı öner |
| 429 | Broker rate limit | `Retry-After` sınırına uyan bounded retry |
| 5xx | Geçici broker hatası | Sınırlı retry, sonra fail-closed |

Hata gövdesi bounded ve makinece okunabilir `code`, `message`, `request_id`
alanlarından oluşur; stack trace, SQL, GitHub URL'si veya secret içermez.

## Broker içi dağıtım

Broker, private GitHub App veya private registry credential'ını yalnızca sunucu
tarafında kullanır. CI tarafından üretilen artifact ve signed release manifest
object storage/registry'den alınır. İstemciye kaynak repo, PAT veya kalıcı
registry credential'ı gönderilmez.

## Operasyon ve kabul

- Grant issuance, denial, expiry ve revoke olayları PII/secret içermeyen audit
  kaydı üretir.
- Signing key HSM/KMS veya erişimi ayrılmış release ortamında tutulur; public key
  rotation için key-id ve overlap penceresi tanımlanır.
- Revocation ve mirror/air-gapped aktarım akışları ayrı runbook ve tatbikatla
  doğrulanır.
- Contract test; valid grant, mismatch, expiry, oversized response, redirect,
  401/403/429/5xx ve token redaction senaryolarını kapsar.

Bu belge API implementasyonu değildir. Gerçek broker endpoint'i, OAuth client
registration'ı, registry hesabı ve signing key ceremony'si sağlanmadan production
hazır kabulü verilmez.
