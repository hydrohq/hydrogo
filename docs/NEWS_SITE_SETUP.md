# HydroGo News Site Setup & Deployment Guide

Bu rehber, HydroGo ile bağımsız ve Core kodunu içermeyen bir **Haber Merkezi / Yayıncılık (News)** Client Edition projesi oluşturma, lisanslama, geliştirme ve devreye alma adımlarını açıklar.

---

## 1. Proje Oluşturma (Scaffolding)

Aşağıdaki komutla yeni bir haber sitesi projesi oluşturun:

```powershell
hydro new haber-merkezi --preset=news --database=sqlite --profile=compact --git
```

Veya etkileşimli modda çalıştırın:

```powershell
hydro new
# Proje adı: haber-merkezi
# Preset: news
# Veritabanı: sqlite
# Profil: compact
# Sunum tipi: integrated
# Demo içerik: y
# Git başlatılsın mı?: y
```

Oluşturulan dizin yapısı:
```text
haber-merkezi/
├── config.yaml          # Çalışma zamanı yapılandırması
├── hydrogo.lock         # Proje kimliği, Core sürümü ve özellik kilitleri
├── README.md            # Proje başlangıç kılavuzu
├── PROJECT_STATUS.md    # Durum ve hazırlık listesi
├── TASKS.md             # Proje görevleri
├── LICENSE              # Client Edition lisans taslağı
├── NOTICE               # 3. parti lisans bildirimleri
├── docs/                # Mimari, operasyon ve E2E kılavuzları
├── themes/
│   └── newsroom/        # Tema şablonları (theme.json, layouts/, static/)
├── custom/              # Müşteriye özel Go eklenti ve kancaları
├── migrations/          # Müşteriye özel ek veritabanı tabloları
├── seed/                # Başlangıç ve demo verileri
└── e2e/
    └── manifest.json    # Playwright E2E test manifestosu
```

> [!NOTE]
> Proje klasöründe HydroGo Core'un kaynak kodu (internal motor paketleri) bulunmaz. Proje lisanslı HydroGo runtime ikili dosyası (`hydro`) ile çalıştırılır.

---

## 2. Lisanslama ve Başlatma

1. Satıcıdan veya lisanslama aracından projenize özel imzalanmış `hydro.lic` dosyasını temin edin ve proje kök dizinine yerleştirin:
   ```powershell
   cd haber-merkezi
   # hydro.lic dosyasının proje kökünde olduğundan emin olun
   hydro license status --license hydro.lic
   ```

2. Veritabanı migrasyonlarını uygulayın:
   ```powershell
   hydro migrate --config config.yaml
   ```

3. Yönetici (Admin) kullanıcısını oluşturun:
   ```powershell
   hydro create-admin --config config.yaml --email editor@haber-merkezi.test --password-stdin
   # Parolanızı girip Enter'a basın
   ```

4. Sunucuyu başlatın:
   ```powershell
   hydro serve --config config.yaml
   ```

Sunucu `http://127.0.0.1:8080` adresinde çalışmaya başlar:
- **Public Haber Odası:** `http://127.0.0.1:8080/public/news/tr/html`
- **Admin Studio Paneli:** `http://127.0.0.1:8080/admin`
- **Üye Girişi:** `http://127.0.0.1:8080/member/login`
- **Sağlık Kontrolü:** `http://127.0.0.1:8080/healthz`

---

## 3. Ayrı Git Repository'sine Bağlama

Oluşturulan proje bağımsız bir Git deposu olarak versiyonlanır:

```powershell
cd haber-merkezi
git add .
git commit -m "Initial news project scaffold"
git remote add origin https://github.com/your-org/haber-merkezi.git
git branch -M main
git push -u origin main
```

> [!WARNING]
> Gizli lisans imzalama anahtarlarını (`license_priv.key`) veya hassas ortam parolalarını depoya eklemeyin. `hydro.lic` ve `config.yaml` güvenle depolanabilir.

---

## 4. Temayı Özelleştirme

`themes/newsroom/` altındaki dosyaları doğrudan düzenleyebilirsiniz:
- `theme.json`: Tema metadata'sı, slotlar ve stil varlıkları.
- `layouts/base.html`: HTML5 iskeleti, meta etiketleri, responsive viewport ve slot yerleşimleri (`header`, `content`, `footer`).
- `static/theme.css`: Renk paleti, tipografi, dark mode CSS değişkenleri ve responsive grid ayarları.

---

## 5. E2E Testlerini Çalıştırma

```powershell
npm test
```
Metadata tabanlı E2E testleri masaüstü, tablet ve mobil görünümde yatay taşma ve kırık görsel denetimlerini otomatik olarak yürütür.
