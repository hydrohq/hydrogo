# HydroGo — Workspace & Engine for Client Editions

HydroGo is a Go-based **product factory and high-performance CMS/BaaS engine**, not a single customer-specific web app. It turns versioned blueprints, modular contracts, and domain policies into deployable, production-ready **Client Editions** for corporate sites, editorial platforms, portals, SaaS backends, and transaction-heavy business products.

The Core remains generic and versioned; specific customer workflows, authorization rules, and presentation layers live in an Edition's `custom/` boundary.

---

## ⚡ Why HydroGo? The Paradigm Shift

Traditional CMS and BaaS stacks force painful trade-offs:
- **WordPress / PHP:** Requires 150+ disk reads per request, consumes 50–100MB RAM per worker, suffers from plugin vulnerabilities, and exposes all plain-text PHP source code to anyone with FTP access.
- **Node.js / PayloadCMS / Strapi:** Demands high-memory VPS/containers ($15–30/month), has 1.5–3s cold-starts, requires heavy JavaScript builds, and cannot run on cheap shared hosting.
- **Laravel / Filament:** Heavy dependency tree, high memory footprint, requires persistent PHP-FPM daemons.
- **PocketBase:** Excellent SQLite BaaS, but lacks a server-rendered (SSR) typed Templ design system, agency client-handover UI, and native CGI shared-hosting deployment.

**HydroGo bridges this gap:** A single compiled binary (~12–16MB) running in **2–4ms cold-start with 8–15MB RAM**, fully dynamic SSR with type-safe Go/Templ, and native CGI support for **$10–15/year cPanel/Plesk shared hosting**, while preserving complete intellectual property code secrecy.

---

## 📊 Feature Comparison Matrix

| Feature / Capability | WordPress | PayloadCMS | Laravel Filament | PocketBase | **HydroGo** |
| :--- | :---: | :---: | :---: | :---: | :---: |
| **Runtime Language** | PHP (Interpreted) | TypeScript / Node.js | PHP (Interpreted) | Go (Compiled) | **Pure Go (Compiled)** |
| **$10/yr cPanel / Shared Hosting** | ⚠️ Slow / Heavy | ❌ Impossible | ⚠️ High Memory | ❌ No native CGI | **✅ Native CGI / FastCGI** |
| **Cold Start / Response Time (TTFB)** | 150ms – 600ms | 800ms – 2500ms | 100ms – 350ms | 5ms – 20ms | **2ms – 10ms (0ms Static Cache)** |
| **Memory Footprint (Per Request)** | 40MB – 120MB | 120MB – 300MB | 30MB – 80MB | 15MB – 30MB | **8MB – 15MB (Compact Profile)** |
| **Code Secrecy & IP Protection** | ❌ Plaintext PHP | ❌ Plaintext JS | ❌ Plaintext PHP | ✅ Compiled Binary | **✅ Stripped Machine Code ELF** |
| **External Dependencies** | MySQL, Apache | Node.js, MongoDB/PG | PHP-FPM, MySQL | None (SQLite) | **Zero (In-Process SQLite)** |
| **Frontend SSR Architecture** | PHP Spaghetti / Theme | React / Next.js SPA | Blade / Livewire | Client-Side SPA / API | **Type-Safe Templ + HTMX** |
| **AI SEO (`/llms.txt` & OG Generator)**| ❌ Eklenti Gerekir | ❌ Custom Code | ❌ Custom Code | ❌ Yok | **✅ Yerleşik Pure Go Motoru** |
| **SMTP Çöküş Kalkanı (Mail Spool)**| ❌ E-postalar Kaybolur | ❌ Yok | ⚠️ Redis Queue | ❌ Yok | **✅ SQLite Spool + MX Fallback** |
| **Client Handoff UX (Zero-Jargon)** | ❌ Teknik Gürültü | ⚠️ Geliştirici Odaklı| ⚠️ Admin Odaklı | ⚠️ DB Viewer Odaklı | **✅ İş Sahibi Görünümü** |

---

## 🏛️ Architecture at a Glance

```text
Blueprint / Edition Manifest / Module Contracts
                         │
                         ▼
                 generated/ Artifacts
                         │      custom/ Policy + Workflow
                         └──────────────┐
                                        ▼
HTTP/JSON/Templ → Application Service → Repository Contract → SQLite / Postgres / MySQL
                                        │
                         Audit · Jobs · Cache · Media · Observability
```

---

## 🚀 Key Modules & Capabilities

### 1. 🌐 Hosting & Runtime Devrimi ($10/Yıl Shared Hosting)
- **Native CGI & FastCGI (`net/http/cgi` & `net/http/fcgi`):** Tek bir binary ile en ucuz cPanel/Plesk paylaşımlı hostingde sıfır VPS maliyetiyle çalışma.
- **Otomatik Statik HTML Disk Önbellekleme (`.htaccess` Passthrough):** Oturum açmamış ziyaretçiler için üretilen sayfalar diske yazılır; sonraki isteklerde Apache doğrudan diskten sunar (**0ms / 0 CPU**).
- **Lazy Web Cron:** CGI ortamında arka plan işlerini (oturum temizliği, haber yayınlama, webhook gönderimi) ziyaretçi isteklerinin kuyruğunda yürüten akıllı cron.
- **Adaptif SQLite Profilleri:** cPanel için 2MB RAM tasarruflu *Compact Profile*, VPS için 64MB RAM ve memory-mapping (mmap) destekli *VPS Profile*.

### 2. 🎨 Tip-Güvenli Templ Tasarım Sistemi (`modules/designsystem`)
- **Bento Grid & Blok Düzen Motoru:** Apple/Linear tarzı modern asimetrik bloklar.
- **İnteraktif Bileşenler:** SSS Akordiyonu (otomatik `Schema.org FAQPage` JSON-LD ile), Before/After Görsel Karşılaştırma Slider'ı, Hafif Video Oynatıcı, Görsel Bilgi Noktaları (Hotspots), Animasyonlu Sayaçlar (`IntersectionObserver`).
- **Mikro-Etkileşimler:** Panoya Kopyalama (`ui.CopyButton`), Form başarı anında Konfeti kutlaması, Akıllı Kaybolan/Beliren Yapışkan Başlık (Smart Sticky Header), Saf CSS Masonry Galeri.

### 3. 📝 Form Motoru, Lead CRM & Dayanıklı E-Posta (`modules/forms` & `modules/mailer`)
- **Form Builder & Gelen Kutusu (Inbox):** Dinamik alanlar, honeypot ve Cloudflare Turnstile spam tuzağı.
- **Çift Gönderim Kilidi (Idempotency):** Mükerrer form gönderimlerini engelleyen istek kilidi.
- **Otomatik Teyit E-Postası (Autoresponder):** `{{name}}`, `{{ticket_code}}` dinamik etiketli şık HTML e-postaları.
- **E-Posta Yazım Hatası Önerisi:** `@gmai.com`, `@hotmial.com` gibi hataları anında tespit edip düzelten mikro UX.
- **Dayanıklı SQLite Mail Spooler:** Müşterinin SMTP sunucusu çökse bile formları kaybetmeyip üssel geri çekilme (exponential backoff) ile yeniden deneyen kuyruk.
- **Zero-Config MX Fallback:** SMTP hesabı tanımlanmadığında doğrudan alıcı DNS MX kayıtları üzerinden acil durum e-posta teslimatı.

### 4. 📈 SEO, AI Arama & Gizlilik Odaklı Analitik (`modules/seo` & `modules/analytics`)
- **Pure Go Dinamik OpenGraph (OG) Görsel Motoru:** 1200x630 sosyal paylaşım kartlarını Node.js/Chromium olmadan Pure Go (`image/draw` + `goregular.TTF`) ile anında üreten ve önbellekleyen motor.
- **AI Arama Motoru İndeksi (`/llms.txt` & `/llms-full.txt`):** ChatGPT, Claude ve Perplexity için optimize edilmiş temiz metin fihristi.
- **Zero-Cookie Yerel Analitik:** GDPR/KVKK uyumlu, ziyaretçiye çerez bırakmayan, sunucu tarafında sayfa görüntüleme ve referans kaynağı ölçümü.
- **Dinamik Sitemap Bölücü:** 50.000+ URL destekli otomatik `sitemap-index.xml` ve kategori parçalama.
- **Performans:** ETag 304 Not Modified, Pure Go Brotli & Gzip sıkıştırma, `sync.Pool` sıfır bellek tüketimli SSR render.

### 5. 🛡️ Kurşun Geçirmez Güvenlik & Dayanıklılık (`internal/security`)
- **Tarpit Progressive Sleep Savunması:** Giriş ekranında hatalı şifre denemelerine artan gecikme (1s -> 2s -> 4s -> 8s) ekleyerek brute-force botlarını CPU harcamadan kilitleme.
- **TOTP İki Adımlı Doğrulama (2FA):** Google Authenticator / 1Password uyumlu RFC 6238 QR kodlu doğrulama.
- **Anlık IP / CIDR Engelleme (`hydro ban`):** Bellek ve veritabanı destekli anında 403 Forbidden kalkanı.
- **SQLite Açılış WAL Kurtarma:** Sunucu ani kapandığında yetim `.db-wal` kilitlerini otomatik temizleyen init katmanı.
- **Disk Kotası Koruma Kalkanı (Quota Guard):** Disk %90+ dolduğunda veritabanı bozulmasını önlemek için yüklemeleri nazikçe durduran koruma.
- **Tek Komutla Geri Alma (`hydro rollback`):** Şema göçü öncesi otomatik anlık yedek alan ve tek komutla eski versiyona dönen kurtarma aracı.
- **Değiştirilemez Denetim Günlüğü (Immutable Audit Log):** Yönetici işlemlerinin manipüle edilemeyen tarihsel kaydı.

### 6. 💼 İşletme Sahibi Odaklı Admin Studio (`internal/adminstudio`)
- **Sıfır Teknik Jargon:** Müşteriye SQL hatası veya UUID göstermeyen iş odaklı arayüz (`docs/INFORMATION_HIERARCHY.md`).
- **Görsel Çeviri Matrisi (i18n):** Paneldeki tüm sabit Türkçe/İngilizce metinleri arayıp tek tıkla düzenleme ve veritabanına kaydetme.
- **Canlı Görsel Diff:** İçerik revizyonları arasındaki farkları Git tarzı yeşil/kırmızı gösteren karşılaştırma aracı.
- **Kayan Toplu İşlem Çubuğu (Bulk Action Toolbar):** Tablolarda çoklu seçimle Excel'e aktarma, okundu işaretleme ve silme.
- **CSV / XLSX İçe Aktarma Sihirbazı:** Sütun eşleştirme ve önizleme ile eski verileri tek tıkla aktarma.
- **Pazartesi Sabahı Yönetici Özeti:** Her Pazartesi 09:00'da işletme sahibine haftalık performans özeti atan bülten motoru.
- **Ajans Destek Kartı & Video Rehber Sekmesi:** Müşterinin teknik destek almasını sağlayan panel içi iletişim kutusu ve video eğitim alanı.

---

## 🛠️ Global CLI Kullanımı

HydroGo komutları Linux, macOS ve Windows ortamlarında **aynı evrensel sözdizimiyle** çalışır:

```bash
# Global Kurulum
go install ./cmd/hydro

# Yeni Proje Başlatma (Şablon Seçenekleri: corporate, news, agency, saas, minimal)
hydro new my-client-site --preset=corporate
cd my-client-site
hydro project check

# Geliştirme Modu (Canlı Yenileme / Hot-Reload)
hydro dev

# Veritabanı ve Teşhis
hydro migrate                         # Bekleyen migration'ları uygula
hydro doctor                          # Sistem sağlığını tara
hydro doctor --fix                    # SQLite kilit ve izinlerini otomatik onar
hydro rollback                        # Son göç öncesi yedeğe geri dön

# Üretim Sunucusu
hydro serve                           # Standalone HTTP sunucusu
hydro serve --cgi                     # cPanel / Plesk CGI modu
hydro ban 198.51.100.4                # Saldırgan IP'yi engelle
hydro clean --demo                    # Demo sahte verileri temizle
```

---

## 📦 Dağıtım Seçenekleri

### 1. Paylaşımlı Hosting (cPanel / Plesk — $10/Yıl)
```bash
make build-cgi
```
Üretilen `hydro.cgi` ve `deploy/hosting/.htaccess` dosyalarını FTP ile `public_html/` dizinine yükleyin. **Tam dinamik site ve yönetim paneli anında yayında!**

### 2. Standalone VPS / Container (Docker / Caddy)
```bash
docker compose -f deploy/docker-compose.prod.yml up -d
```

---

## 🧪 Testler ve Doğrulama

Tüm çekirdek ve modül testlerini çalıştırmak için:

```bash
go test ./...
```

Eşzamanlılık ve Race denetimi:

```bash
go test -race ./...
```

---

## 📄 Lisans ve Mülkiyet

HydroGo kapalı kaynak ve tescilli bir platformdur. Lisanslama ve ticari kullanım şartları [`PROPRIETARY_NOTICE.md`](PROPRIETARY_NOTICE.md) ve [`docs/COMMERCIAL_LICENSING.md`](docs/COMMERCIAL_LICENSING.md) belgelerinde belirtilmiştir.
