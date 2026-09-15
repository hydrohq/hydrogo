# Hydro SDK ve private HydroGo runtime dağıtım planı

**Durum:** Sözleşme ve güvenli yerel artifact zinciri uygulanıyor; broker,
login, imzalı release ve installer yayınları açık. Ayrıntılı durum:
[`IMPLEMENTATION_STATUS.md`](IMPLEMENTATION_STATUS.md).  
**Amaç:** Linux, Windows ve macOS’ta Node.js benzeri kurulabilen public `hydro` CLI ile private HydroGo Core runtime’ını kaynak kodu dağıtmadan kullanmak.

## Temel karar

Public SDK; CLI, generator, starter’lar, contract’lar, frontend adapter’ları ve geliştirici araçlarını içerir. HydroGo Core ve ticari modüller private kalır; istemciye kaynak kodu kopyalanmaz. Runtime, imzalı binary/artifact olarak yetkilendirilmiş kanaldan indirilir.

Bu yaklaşım mevcut Core izolasyonu, `hydrogo.lock`, edition manifest ve offline lisans kararlarıyla uyumludur. SDK, Go package sınırı veya `internal` görünürlüğünü bir güvenlik sandbox’ı olarak sunmaz; aynı binary içindeki trusted code güvenilir kabul edilir.

## Kullanıcı deneyimi

```text
hydro login
hydro project new haber-sitesi --preset=news
cd haber-sitesi
hydro dev
hydro resource new articles
hydro generate
hydro migrate
```

## Dağıtım bileşenleri

### Public Hydro SDK

- `hydro` CLI
- Project/resource generator
- Starter template’leri
- OpenAPI/TypeScript üreticileri
- React/HTMX/vanilla adapter’ları
- `doctor`, `check`, `schema diff`, `api diff`
- Local dev proxy ve test runner

### Private artifact service

- Platforma göre runtime binary
- Module/kit artifact’ları
- İmzalı release manifesti
- Kısa ömürlü download URL/token
- License ve edition entitlement kontrolü

### HydroGo runtime

- Core + seçilmiş modüller
- Migration ve compatibility metadata
- Güvenilir public key ile binary/manifest doğrulaması
- Offline `hydro.lic` desteği

## Private GitHub erişimi

`git clone` veya `git ls-remote` doğrudan ana yetkilendirme mekanizması yapılmaz. Git komutu yalnızca kurumsal geliştirici modunda teşhis amacıyla kullanılabilir. Çünkü repo erişimi; lisans, sürüm entitlement’ı ve artifact bütünlüğü anlamına gelmez. Ayrıca token sızıntısı, SSO, proxy, rate limit ve belirsiz hata ayrımı sorunları oluşturur.

Önerilen akış:

```text
hydro login
    ↓
Hydro account/license service
    ↓
GitHub App veya private artifact registry
    ↓
kısa ömürlü download URL
    ↓
checksum + imza doğrulama
    ↓
lokal runtime cache
```

GitHub App installation token veya sunucu tarafında saklanan credential kullanılır; kullanıcı bilgisayarına kalıcı GitHub PAT gömülmez. Kaynak repo yerine CI tarafından üretilmiş release artifact indirilir.

## Platform kurulumları

- Windows: MSI, WinGet, imzalı `.exe`, Windows Credential Manager.
- macOS: Homebrew tap, notarized arm64/amd64 veya universal binary, Keychain.
- Linux: `.deb`, `.rpm`, tarball; amd64/arm64; checksum ve signature zorunluluğu.
- Her kurulumdan sonra `hydro doctor` PATH, platform, runtime, lisans ve imza durumunu kontrol eder.

## Lock ve uyumluluk

Proje `hydrogo.lock` içinde SDK, runtime, contract, edition, modül seti ve hash’leri kilitler. CLI şu kontrolleri yapar:

- SDK/runtime/module uyumu
- License ve edition entitlement
- Platform/CPU uyumu
- Migration ve API compatibility
- İmza/checksum
- Offline grace period

## Güvenlik kuralları

- Private key yalnızca release/signing ortamında tutulur.
- İstemciye private repo token’ı yazılmaz.
- Artifact indirme URL’si kısa ömürlü olur.
- Binary ve manifest doğrulanmadan çalıştırılmaz.
- Download, login ve license log’ları sırları içermez.
- Air-gapped kullanım için imzalı offline lisans ve manuel artifact aktarımı desteklenir.
- License bitişi public siteyi beklenmedik biçimde kesmez; entitlement/update davranışı açıkça tanımlanır.

## Uygulama sırası

1. SDK/runtime/contract sürüm modelini ve lock formatını kesinleştir.
2. CI’da cross-platform imzalı runtime artifact üret.
3. Artifact manifest ve checksum doğrulamasını CLI’a ekle.
4. `hydro login` için OAuth/device flow ve kısa ömürlü token servisini kur.
5. GitHub App/private registry entegrasyonunu broker arkasına al.
6. Windows/macOS/Linux installer’larını yayınla.
7. Offline lisans ve air-gapped runbook’u doğrula.
8. Revocation, rotation, mirror ve rollback tatbikatlarını yap.

## Açık kararlar

- Hosted Hydro hesabı mı, yalnızca GitHub organization hesabı mı?
- Artifact registry GitHub Releases, GHCR veya ayrı object storage mı?
- Ticari entitlement ile repository membership nasıl ayrılacak?
- Tam offline müşterilerde artifact güncellemesi nasıl aktarılacak?
- Runtime binary’si tek dosya mı, module bundle mı olacak?

Bu kararlar kabul edilmeden public SDK release’i ve müşteriye dağıtım sözü verilmez.
