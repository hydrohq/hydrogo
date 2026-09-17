# ADR-0028: Core isolation and commercial licensing

## Status
Superseded by [ADR-0137](0137-commercial-license-issuer-boundary.md) for the issuer/runtime CLI boundary. The core isolation and offline verification decisions remain valid.

## Context

HydroGo platformunun kurumsal müşterilere ve ajanslara teslim edilmesinde, çekirdek iş mantığının (**HydroGo Core**) müşteriye özel tema, şablon, migrasyon ve konfigürasyon dosyalarından (**Client Edition**) izole edilmesi gerekmektedir. 

Üretilen müşteri projelerinde Core kaynak kodunun bulunmaması, projenin kaynak kodu olarak serbestçe depolanabilmesi ancak çalıştırılabilmesi ve derlenebilmesi için geçerli bir ticari lisans gerektirmesi hedeflenmektedir.

Ayrıca Core dağıtımı için 3 alternatif mimari model araştırılmıştır:
1. **Private Go Module:** Özel erişim kontrollü Git/GOPROXY üzerinden tüketim.
2. **Lisanslı HydroGo Binary:** Önceden derlenmiş, konfigürasyon ve tema yükleyebilen tekil ikili dosya.
3. **Hosted / SaaS Runtime:** Tamamen sağlayıcı altyapısında barındırılan çok kiracılı veya yönetilen çalışma ortamı.

## Decision

1. **Core İzolasyonu:** Oluşturulan projeler hiçbir HydroGo Core kaynak kodunu içermez. Generator yalnızca Client Edition (`extension/v1`), tema (`themes/`), config (`config.yaml`), migrasyonlar, frontend ve E2E testlerini üretir.
2. **Hibrit Dağıtım Çözümü (Önerilen Varsayılan):** Core, standart kullanımda önceden derlenmiş **Lisanslı Binary** veya kurumsal projelerde **Private Go Module** olarak dağıtılır.
3. **Kilit Dosyası (`hydrogo.lock`):** Üretilen projenin kök dizininde yer alan `hydrogo.lock` dosyası Core sürümünü, generator sürümünü, edition tipini, preset ayarını, proje kimliğini ve gerekli özellik kümesini bağlar.
4. **Ed25519 İmzalı Çevrimdışı (Offline) Lisanslama:**
   - Lisans dosyası (`hydro.lic`) Ed25519 algoritmasıyla merkezde imzalanır.
   - Doğrulama için gereken Public Key uygulamaya gömülüdür; Private Key asla dağıtılmaz veya müşteri sunucularına iletilmez.
   - Doğrulama akışı `customer`, `project_id`, `edition`, `features`, `expires_at` ve `grace_period_days` kontrollerini içerir.
   - Lisans süresi dolduğunda kesintiyi önlemek amacıyla yapılandırılabilir **Grace Period (Tolerans Süresi)** devreye girer ve durum loglarına yansıtılır.
   - İhlal veya uyuşmazlık durumlarında operatöre net ve insan okunabilir Türkçe/İngilizce hata mesajları döndürülür.
5. **CLI Lisans Yönetimi:** Vendor-only `hydro-issuer` içerisine `issue` ve `keygen`; müşteri `cmd/hydro` içerisine yalnızca `verify` ve `status` komutları yerleştirilmiştir. Ayrıntı ve superseding karar için [ADR-0137](0137-commercial-license-issuer-boundary.md) geçerlidir.
6. **Project Scaffold:** `hydro new PROJECT --preset=...` creates a client-owned scaffold with no Core source. The scaffold contains config, lock metadata, a minimal theme, project migration/custom placeholders and E2E manifest.
7. **Runtime Enforcement:** When `license.required: true` is configured, `serve` requires `hydro.lic`, the configured project/edition/features, and the official binary's trusted public key. Development builds without an injected vendor key fail closed for licensed projects.

## Resource & Security Boundaries

- **Kriptografik Sınırlar ve Gerçekçilik:** Lisans dosyalarının Ed25519 ile imzalanması verinin kurcalanmasını (tamper) kesin olarak tespit eder. Ancak istemci makinesinde çalışan derlenmiş ikili dosyaların (binary) tersine mühendislikle incelenmesi teknik olarak mümkündür. "Kodun kopyalanması teknik olarak imkânsızdır" gibi gerçekçi olmayan iddialarda bulunulmaz; teknik imza koruması kurumsal sözleşmeler ve yasal mevzuat ile birlikte değerlendirilir.
- **Kaynak Tüketimi:** Lisans denetimi başlangıç anında bir kez ve bellek içinde gerçekleşir. Harici bir "phone-home" ağına veya lisans sunucusuna ihtiyaç duyulmadığından Compact profilde sıfır ağ gecikmesi ve ihmal edilebilir CPU/bellek maliyeti oluşturur.
- **Hukuki Statü:** Lisans metinleri bir mühendislik taslağıdır ve "lawyer review required" şartına tabidir.
- **Key Trust:** The `public_key` field in a license is informational only. Runtime verification requires the vendor key injected into the official build; accepting a self-advertised key would allow self-signed licenses.

## Consequences

- Müşteri projeleri temiz, hafif ve Core'dan bağımsız kaynak havuzlarında yönetilebilir.
- Müşteri projeleri Core olmadan paylaşılabilir ancak geçerli `hydro.lic` ve lisanslı Core olmadan çalıştırılamaz.
- Dağıtık ve air-gapped ortamlarda internet gereksinimi olmadan güvenli lisans denetimi sağlanır.
