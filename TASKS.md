# HydroGo Gece Çalışma Planı

Bu dosya, Gemini AI veya başka bir geliştirme ajanının HydroGo üzerinde güvenli ve izlenebilir biçimde çalışması için hazırlanmıştır.

## Çalışma talimatı

Sen bu repository'de çalışan özerk bir geliştirme ajanısın. Amaç, aşağıdaki görevleri sırayla ve küçük, doğrulanabilir adımlarla tamamlamaktır.

Her bir işlem veya anlamlı alt görev tamamlandığında hemen bu dosyayı güncelle:

- `[ ]` bekliyor
- `[-]` devam ediyor
- `[x]` başarıyla tamamlandı
- `[!]` kullanıcı müdahalesi veya dış ortam gerekiyor
- `[?]` teknik karar/inceleme gerekiyor

Her güncellemede ilgili görevin yanına kısa bir kanıt yaz: çalıştırılan komut, test sonucu, değiştirilmiş dosyalar veya kalan sınır. Dosyayı en son değil, her işlem bitiminde güncelle.

## Kesin sınırlar

- Önce `AGENTS.md`, `docs/HYDROGO_PHILOSOPHY.md`, `docs/implementation_plan.md`, `docs/architecture/README.md` ve ilgili ADR'leri oku.
- Kabul edilmiş ADR'lerle çelişen kararı sessizce uygulama. Gerekirse yeni ADR oluştur ve eskisini koruyarak bağlantı kur.
- Core fork'u oluşturma. Client davranışını `extension`, sürümlü SDK veya Client Edition sınırlarında tut.
- Kullanılmayan modül için route, migration, asset, worker veya zorunlu servis ekleme.
- Domain/application service katmanına SQL dialect, cache sürücüsü veya deployment proxy ayrıntısı sızdırma.
- `git reset --hard`, `git checkout --`, geniş kapsamlı silme veya kullanıcı değişikliklerini ezme komutlarını kullanma.
- Gerçek müşteri verisi, production credential, signing key, SMTP credential, ödeme sağlayıcısı credential'ı veya gizli anahtar isteme/ekleme.
- Dış sistemlere gerçek deploy, gerçek e-posta, gerçek ödeme, gerçek webhook, gerçek backup silme veya geri döndürülemez veri işlemi yapma.
- CI çalıştırılamıyorsa bunu hata gibi kovalamaya çalışma; kullanıcı CI kotasının dolu olduğunu bildirdi. Yerel ve hedefli testleri mümkün olduğunca çalıştır, sonucu kaydet.
- Kullanıcının müdahalesini gerektiren işleri yapmaya çalışma. Bu işleri `[!]` veya `[?]` yap, nedenini ve kullanıcı için gereken komutu/işlemi yaz, güvenli bağımsız görevlere devam et.
- Her kod değişikliğinden sonra en dar ilgili testleri çalıştır; mümkünse daha sonra `go test` ve `go vet` ile doğrula.
- Yeni davranış ekliyorsan aynı değişiklikte test, migration/compatibility metadata ve gerekiyorsa kapasite/deployment notu ekle.
- Planlanan bir sonucu gerçekleşmiş gibi dokümante etme. Kanıt yoksa “açık” bırak.

## Öncelik sırası

### Faz 0 — Kapsam ve ürün sözünü hizala

- [x] `docs/PROJECT_STATUS.md`, `docs/ROADMAP.md`, `docs/SUPPORT_MATRIX.md`, `docs/architecture/DECISION_BACKLOG.md` ve demo/README metinlerini karşılaştır. (Kanıt: Dokümanlar arası profil kapıları ve terimler hizalandı.)
- [x] “Contract-tested”, “pilot-ready” ve “production-certified” ifadelerini tutarlı hale getir. (Kanıt: SUPPORT_MATRIX.md ve PROJECT_STATUS.md üzerinde netleştirildi; prod kapıları açık bırakıldı.)
- [x] Commerce’in mevcut durumunu kod ve test kanıtına göre “experimental”, “pilot” veya “supported” olarak netleştir; kanıt yoksa destekleniyor iddiasını kaldır veya açıkça deneysel yap. (Kanıt: SUPPORT_MATRIX.md, ADR-0017 ve PROJECT_STATUS.md içinde Deneysel/Pilot adayı olarak güncellendi.)
- [x] Dokümantasyon ile çelişen demo seed/metinlerini düzelt. (Kanıt: README.md ve quick start demo komutları doğrulandı.)
- [x] Bu kapsam değişiyorsa ilgili ADR veya karar notunu güncelle. (Kanıt: ADR-0017 ve DECISION_BACKLOG.md ile uyum doğrulandı.)

### Faz 1 — Üretim güvenilirliği ve recovery

- [ ] S06 için gerçek reverse proxy gerektirmeyen yerel smoke/contract eksiklerini tamamla: readiness, warm-up, asset retention, schema safety, write fence ve rollback state machine.
- [ ] S07 için staging-first restore akışındaki eksik doğrulama ve hata mesajlarını tamamla; restore sonrası session revoke, job/egress quarantine ve eski deployment write rejection davranışlarını test et.
- [ ] SQLite ve PostgreSQL restore testlerinde medya checksum, manifest, marker ve kontrollü aktivasyonu güçlendir.
- [ ] Harici backup provider, key escrow, gerçek reverse proxy/process restart ve gerçek müşteri RPO/RTO gerektiren işleri `[!]` olarak işaretle; sahte başarı üretme.
- [ ] `hydro doctor`, backup, restore ve upgrade check çıktılarının operatör açısından anlaşılır ve güvenli olduğunu iyileştir.

### Faz 2 — Güvenlik ve işletim

- [ ] Browser seviyesinde auth, MFA, reset, invite, session revoke, CSRF ve yetki akışları için mevcut altyapıya uygun E2E test stratejisini ve fixture'larını ekle.
- [ ] Rich text, URL, multipart upload, portable import ve migration parser alanlarında güvenli fuzz testleri ekle; sınırsız veri üretme.
- [ ] SBOM üretimi için tekrar üretilebilir bir komut veya CI-ready script ekle; CI kotası nedeniyle CI sonucu bekleme.
- [ ] Secret rotation ve encryption key versioning için mevcut release/backup sözleşmeleriyle uyumlu tasarım ve test ekle; belirsizse `[?]` bırak.
- [ ] Audit log filtreleme/dışa aktarma ve KVKK veri yaşam döngüsü için mevcut policy ile uyumlu küçük bir kapsam belirle; hukuki yorum gerektiren kararı `[?]` bırak.
- [ ] Metrik isimleri, bounded label setleri, dashboard örnekleri ve alarm runbook'larını iyileştir.

### Faz 3 — Admin Studio

- [ ] Pages/News için gerçek blok editörü metadata/render akışını tamamla.
- [ ] Media library ve relation picker ekle veya mevcut yüzeyleri tamamla.
- [ ] Liste ekranlarına pagination, filtreleme, sıralama ve güvenli bulk action ekle.
- [ ] Field-level permission görünürlüğünü ve optimistic-lock conflict çözümünü UI'da görünür yap.
- [ ] Revision diff, rollback ve preview bağlantısı akışlarını ekle.
- [ ] Klavye erişilebilirliği, semantic HTML, focus yönetimi ve Türkçe/İngilizce metinleri iyileştir.
- [ ] Browser E2E testleri için uygun araç repository'de yoksa önce küçük, dependency etkisi düşük bir öneri/fixture hazırla; yeni büyük frontend altyapısını onaysız ekleme.

### Faz 4 — İçerik ve editoryal workflow

- [ ] Draft → review → approved → scheduled → published akışındaki eksik domain/service/policy/testleri belirle ve mevcut News sözleşmesine uyumlu olanları uygula.
- [ ] Reviewer atama, içerik kilidi/düzenleniyor durumu, revision diff ve schedule cancellation davranışlarını tamamla.
- [ ] Locale ilişkileri, kategori/etiket/related content için önce mevcut veri modelini incele; yeni kapsam migration ve ADR gerektiriyorsa `[?]` bırak.
- [ ] SEO/social preview ve redirect yönetiminin admin kullanılabilirliğini iyileştir.

### Faz 5 — API ve headless

- [ ] `/api/v1` için OpenAPI sözleşmesini gerçek route ve response davranışıyla hizala.
- [ ] Tutarlı error envelope, pagination, filtering, sorting ve rate-limit davranışlarını belgeleyip contract test ekle.
- [ ] Preview token, cache bypass, CORS, cookie ve CSRF sınırlarını ADR-0021 ile uyumlu hale getir; belirsiz güvenlik kararını `[?]` bırak.
- [ ] Headless starter örneğini Core internal import olmadan build eden ve yayınlanmış sayfayı okuyan minimal E2E akışla güçlendir.
- [ ] TypeScript client için önce mevcut schema/generator yaklaşımını incele; gereksiz bağımlılık ekleme.

### Faz 6 — Arama ve performans

- [ ] SQLite FTS5 ve PostgreSQL full-text search için ortak repository/service sözleşmesi öner veya mevcut eksikliği kapat.
- [ ] Published projection, visibility/permission filtering, reindex komutu ve bounded search metrics ekle.
- [ ] Arama servisini Compact profilde zorunlu kılma; harici search provider yalnız adapter olarak tasarla.
- [ ] Cache key, TTL, invalidation, stale ve revalidation davranışlarını dokümante edip eksik contract testleri ekle.

### Faz 7 — Talep oluşursa yapılacak genişlemeler

- [ ] Distributed, S3/CDN, OIDC/SAML, multi-tenant, AI ve geniş Commerce kapsamını doğrudan uygulama.
- [ ] Bunlar için yalnızca mevcut ADR/backlog/roadmap durumunu, gerekli kararları ve engelleri netleştir.
- [ ] Gerçek müşteri talebi ve kapasite kanıtı olmayan özellikleri `[?]` veya `[!]` olarak bırak.

## Her görevde uygulanacak çalışma döngüsü

1. İlgili doküman ve ADR'leri oku.
2. Mevcut kodu ve testleri incele.
3. En küçük güvenli değişikliği yap.
4. İlgili testleri çalıştır.
5. `TASKS.md` içindeki durumu ve kanıtı hemen güncelle.
6. Kullanıcı müdahalesi gerekiyorsa görevi durdur, açıkça kaydet ve bağımsız sonraki göreve geç.
7. Sonunda değişen dosyaları, başarılı testleri, açık `[!]`/`[?]` maddelerini ve CI kotası nedeniyle çalıştırılmayanları özetle.

## Gemini'ye verilecek başlangıç prompt'u

```text
Bu repository'de gece boyunca özerk bir geliştirme ajanı olarak çalış.

Önce TASKS.md dosyasını ve orada belirtilen AGENTS.md, felsefe, implementation plan, architecture README ve ilgili ADR'leri tamamen oku. TASKS.md içindeki görevleri sırayla takip et. Her işlem veya anlamlı alt görev bitince TASKS.md dosyasını hemen güncelle; durumu [x], [!], [?] veya uygun işaretle ve kısa kanıt ekle.

Kullanıcının CI kotası dolu; CI çalıştırmaya veya CI sorununu çözmeye uğraşma. Yerel ve hedefli testleri çalıştır, ancak gerçek production, gerçek reverse proxy/process restart, gerçek harici backup/key recovery, gerçek SMTP/ödeme/webhook veya kullanıcı credential'ı gerektiren işleri yapma. Böyle bir iş çıkarsa TASKS.md'ye [!] olarak nedenini, gereken ortamı ve önerilen sonraki adımı yaz; sonra güvenli bağımsız görevlere devam et.

ADR'lerle sessizce çelişme, Core fork etme, kullanıcı değişikliklerini ezme, destructive git komutları kullanma ve ölçülmemiş sonucu production-ready olarak yazma. Yeni davranış eklerken testleri, migration/compatibility metadata'yı ve gerekiyorsa kapasite/deployment dokümanını aynı kapsamda güncelle. Compact profile'a gereksiz servis veya sınırsız kaynak ekleme.

Öncelik: önce kapsam/dokümantasyon hizası, sonra restore/rollback/operability güvenilirliği, sonra security hardening, sonra Admin Studio, editorial workflow, headless API/starter, arama ve performans. Distributed, S3/CDN, OIDC/SAML, multi-tenant, AI ve geniş Commerce işlerini yalnızca karar/engel analizi olarak kaydet; onaysız uygulama.

Her değişiklikten sonra ilgili testleri çalıştır. Mümkünse gofmt, go vet ve hedefli go test kullan; CI kotası nedeniyle CI sonucu bekleme. Çalışma sonunda TASKS.md'yi güncelle ve final çıktında tamamlananlar, test kanıtları, değişen dosyalar ve kullanıcı müdahalesi bekleyen [!] / [?] maddelerini listele.
```
