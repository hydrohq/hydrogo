# Gerçek dünya production hazırlık roadmap'i

**Tarih:** 2026-09-18  
**Amaç:** HydroGo'nun kontrollü pilotlardan, ölçülebilir ve desteklenebilir
production dağıtımlarına geçmesi.

Bu belge bir özellik listesi değil, production iddiası için gereken kanıtların
ve operasyonel kararların sıralı planıdır. Bir faz, kod yazılmış olmasıyla değil,
kabul kriterleri ölçülüp raporlandığında tamamlanır. Yerel testlerin geçmesi,
gerçek müşteri ortamı kanıtının yerine geçmez.

## Başlangıç çizgisi

Bugün güçlü bir teknik temel mevcuttur: SQLite/PostgreSQL sözleşmeleri, auth/MFA,
CSRF, audit, idempotency, job leasing, backup/restore akışları, modül izolasyonu,
generator, binary sağlık kontrolleri ve kapsamlı güvenlik/test kapıları çalışır.

Henüz genel production sertifikasyonu yoktur. Commerce deneysel/pilot seviyededir;
Distributed, gerçek S3/CDN failover, gerçek proxy rollback, müşteri verisiyle
restore, RPO/RTO ve kapasite kanıtları açıktır. Bu başlangıç durumu
[`PROJECT_STATUS.md`](PROJECT_STATUS.md) ve
[`OPEN_GATES_AND_OPERATOR_ACTIONS.md`](OPEN_GATES_AND_OPERATOR_ACTIONS.md) içinde
ayrıntılı tutulur.

## Faz 0 — Kapsam ve release politikası

**Hedef:** Her dağıtımın neyi desteklediğinin yazılı ve tekrarlanabilir olması.

- Compact, Standard ve Distributed için desteklenen Go/OS/DB/Redis/S3 sürümlerini
  kesinleştir.
- Pages/News, Commerce ve gelecekteki modülleri ayrı capability/edition olarak
  tanımla; deneysel özellikleri production varsayılanı yapma.
- `hydrogo.lock`, release manifesti, edition ve lisans feature listesinin tek bir
  release kimliği oluşturduğunu doğrula.
- Sürümleme, deprecation, güvenlik yaması, LTS ve destek/SLA politikasını yayınla.

**Çıkış kriteri:** Güncel [destek matrisi](SUPPORT_MATRIX.md), release checklist'i
ve her profile ait “desteklenir/desteklenmez” beyanı onaylanmış olmalı.

## Faz 1 — Güvenli release ve tedarik zinciri

**Hedef:** Müşteriye tekrar üretilebilir, kaynağı doğrulanabilir ve secretsız paket
teslim etmek.

- Release binary'sini sabitlenmiş toolchain ile reproducible üret.
- Vendor signing key'i ayrı, erişim kontrollü ortamda tut; `hydro-issuer` erişimini
  müşteri runtime'ından ayır.
- SBOM, dependency/vulnerability taraması, CodeQL, action pinleri ve artifact
  provenance çıktısını release'e bağla.
- Binary-only müşteri paketinde Core/modül kaynak kodu, private key ve build secret
  bulunmadığını otomatik kontrol et.
- Windows, Linux ve macOS için imzalı artifact ve checksum yayımla.

**Çıkış kriteri:** Temiz makinede paket kurulumu, lisans doğrulaması, checksum,
tamper reddi ve `hydro version --json` kimlik doğrulaması geçer.

## Faz 2 — Production güvenlik ve kimlik

**Hedef:** İnternete açık yönetim yüzeyini güvenli varsayılanlarla işletmek.

- Gerçek domain ve sertifika arkasında HTTPS termination, secure cookie, HSTS ve
  canonical origin testi yap.
- Secret manager kullan; MFA master key, session/config secret, SMTP ve ödeme
  sırlarını dosya/repository dışında tut.
- İlk admin oluşturma, MFA zorunluluğu, parola reseti, davet, oturum iptali ve
  recovery prosedürlerini runbook'a bağla.
- SMTP teslim, bounce, retry ve alarm akışlarını gerçek sağlayıcıyla doğrula.
- Yetkilendirme, CSRF, rate limit, upload ve hata redaksiyonu için DAST/pentest
  öncesi regression paketini çalıştır.

**Çıkış kriteri:** Threat model'deki P0/P1 bulguları kapalı veya yazılı risk
kabulleriyle onaylı; gerçek SMTP ve HTTPS smoke raporu mevcut.

## Faz 3 — Veri dayanıklılığı ve upgrade

**Hedef:** Veri kaybı veya hatalı release durumunda güvenilir geri dönüş.

- Temiz SQLite ve PostgreSQL hedeflerinde müşteri benzeri dataset restore et.
- Backup, migration, rollback ve media activation akışlarını süre/disk/veri kaybı
  ölçümleriyle tekrarla.
- Gerçek reverse proxy arkasında iki process Blue/Green geçişi, readiness/warmup,
  eski-yeni binary uyumu ve yeni yazı sonrası rollback deneyi yap.
- Harici backup/S3 ve KMS/key escrow kullanılıyorsa erişim kaybı ve anahtar kurtarma
  tatbikatı yap.
- Retention, silme, erasure replay ve session revocation sonuçlarını denetim kaydıyla
  doğrula.

**Çıkış kriteri:** Profile özel RPO/RTO hedefleri ölçülmüş; temiz hedef restore'u,
rollback ve veri bütünlüğü raporları imzalanmış.

## Faz 4 — Gözlemlenebilirlik ve işletim

**Hedef:** Sorunları kullanıcı verisini sızdırmadan fark etmek ve çözmek.

- Health/readiness, metrics, structured log, request ID ve audit korelasyonunu
  merkezi log/metric sistemine bağla.
- Backup, disk, migration, SMTP, dead-letter, queue, DB pool, cache ve readiness
  alarmlarını gerçek notification kanalıyla test et.
- PII/secret redaksiyonu, retention ve erişim yetkisini belgede kesinleştir.
- Incident response, bakım penceresi, rollback, disaster recovery ve müşteri iletişim
  runbook'larını tatbikatla doğrula.

**Çıkış kriteri:** Alarm üretme–müdahale–kapanış tatbikatı ölçülmüş; on-call sahibi,
eskalasyon süresi ve hata bütçesi tanımlı.

## Faz 5 — Performans ve kapasite

**Hedef:** “Çalışıyor” yerine profile ve donanıma bağlı kapasite sözü verebilmek.

- Compact/Standard/Distributed için temsili dataset ve trafik profili oluştur.
- RSS, CPU, DB pool, queue depth, cache hit/miss, disk, p95/p99 latency, hata oranı
  ve worker backpressure ölç.
- Uzun süreli soak, burst, concurrent editor, media derivative, rate-limit, Redis
  outage/partition ve cache invalidation testlerini çalıştır.
- PostgreSQL connection/isolation, Redis persistence/failover ve S3/CDN latency/error
  sınırlarını sağlayıcı bazında kaydet.
- Sonuçları kapasite tablosu, autoscaling/limit önerisi ve bilinen sınırlarla yayımla.

**Çıkış kriteri:** Her satışa açılacak profile ait donanım varsayımı, trafik sınırı,
ölçülmüş p95/p99 ve hata/geri kazanım hedefleri vardır.

## Faz 6 — Ürün ve entegrasyon sertifikasyonu

**Hedef:** Gerçek kullanım senaryolarını güvenilir biçimde desteklemek.

- Pages/News için responsive public/admin E2E; boş, hata, preview, schedule, media
  ve yetki senaryolarını tamamla.
- Commerce için gerçek sandbox hesaplarıyla callback/signature, idempotency, refund,
  timeout, webhook retry ve stok yarışlarını doğrula. Canlı ödeme ancak ayrıca
  onaylanmış production gate sonrası açılır.
- S3/CDN için IAM, KMS, retention, cache purge, failover ve private URL sürelerini
  gerçek sağlayıcıyla doğrula.
- OIDC/SAML, mobile API, multi-tenant, AI ve ileri arama gibi özellikleri ayrı ADR,
  threat model ve kapasite kapısı olmadan çekirdek kapsamına alma.

**Çıkış kriteri:** Her capability için provider/ortam matrisi, E2E raporu ve bilinen
  sınırlamalar yayımlanmış; deneysel modüller açıkça etiketlenmiş.

## Faz 7 — Pilot, destek ve ticari açılış

**Hedef:** Bir gerçek müşteriyle kontrollü işletim ve geri bildirim döngüsü.

- Tek bir Compact veya Standard pilot seç; kapsamı, veri sınıfını ve başarı metriklerini
  yazılı sınırla.
- Go-live öncesi checklist, backup/restore provası, erişim matrisi ve rollback planını
  müşteriyle birlikte imzala.
- Pilot boyunca incident, performans, deploy sıklığı, restore süresi ve kullanıcı
  geri bildirimlerini kaydet.
- Pilot çıkışında destek yükü, dokümantasyon eksikleri ve kırıcı değişiklikleri
  roadmap'e geri besle.

**Çıkış kriteri:** En az bir gerçek pilot, tanımlı süre boyunca ciddi veri/güvenlik
  olayı olmadan işletilmiş; SLA ve destek koşulları kanıtla uyumlu.

## Production ilanı için minimum kapı

Bir profil ancak aşağıdakilerin tamamı sağlanınca “production-ready” olarak
ilan edilebilir:

1. Release binary'si imzalı, reproducible ve secretsızdır.
2. HTTPS, secret yönetimi, MFA, SMTP ve alarm akışları gerçek ortamda doğrulanmıştır.
3. Temiz hedef restore'u, rollback ve veri bütünlüğü raporu vardır.
4. Profile özel kapasite, RPO/RTO ve soak sonuçları yayımlanmıştır.
5. Güvenlik taraması/pentest bulguları kapatılmış veya açık risk olarak onaylanmıştır.
6. Runbook, destek matrisi, sürüm/deprecation politikası ve müşteri sorumlulukları
   günceldir.

Bu kapıların herhangi biri eksikse doğru ifade “teknik olarak kullanılabilir pilot”
veya “production adayı”dır; genel production sertifikasyonu değildir.

## Takip yöntemi

Her madde için kod değişikliği gerekiyorsa ilgili ADR, migration/uyumluluk metadatası,
SQLite/PostgreSQL testleri ve kapasite/dağıtım notu aynı değişiklikte güncellenir.
Gerçek ortam kanıtları `docs/validation/` altında tarih, commit, toolchain, komut,
sonuç ve sınırlamalarla tutulur. Ana teknik sıra ve tarihsel fazlar
[`ROADMAP.md`](ROADMAP.md)'de; operasyonel açık maddeler
[`OPEN_GATES_AND_OPERATOR_ACTIONS.md`](OPEN_GATES_AND_OPERATOR_ACTIONS.md)'ndedir.
