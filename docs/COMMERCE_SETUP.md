# HydroGo Commerce Site Setup & Operations Guide

Bu rehber, HydroGo ile bağımsız bir **E-Ticaret ve Mağaza (Commerce)** Client Edition projesi oluşturma, lisanslama, ödeme yapılandırması ve devreye alma adımlarını açıklar.

---

## 1. Proje Oluşturma

```powershell
hydro new magaza --preset=commerce --database=postgres --profile=standard --git
```

Etkileşimli kullanım:
```powershell
hydro new
# Proje adı: magaza
# Preset: commerce
# Veritabanı: postgres
# Profil: standard
# Sunum tipi: integrated
# Demo içerik: y
# Git başlatılsın mı?: y
```

---

## 2. Yapılandırma (`config.yaml`) ve Ödeme Sağlayıcıları

`config.yaml` içerisinde veritabanı DSN ve Commerce modül ayarlarını kontrol edin:

```yaml
profile: standard
modules:
  - pages
  - commerce
  - media
  - members
  - site
http:
  address: "127.0.0.1:8080"
  public_origin: "https://magaza.example.com"
database:
  dialect: postgres
  dsn: "postgres://postgres:password@127.0.0.1:5432/magaza_db?sslmode=disable"
media:
  dir: "data/media"
theme_dir: "themes"
theme: "commerce"
license:
  required: true
  file: "hydro.lic"
  project_id: "proj-magaza"
  edition: "growth"
  features: ["pages", "commerce", "media", "members", "site"]
```

### Ödeme Sağlayıcı Çevre Değişkenleri

Ödeme sağlayıcıları production ortamında ortam değişkenleri üzerinden bağlanır:

```powershell
# Stripe
$env:HYDRO_STRIPE_SECRET_KEY="sk_live_..."
$env:HYDRO_STRIPE_WEBHOOK_SECRET="whsec_..."

# iyzico
$env:HYDRO_IYZICO_API_KEY="api_key_..."
$env:HYDRO_IYZICO_SECRET_KEY="secret_key_..."
$env:HYDRO_IYZICO_BASE_URL="https://api.iyzipay.com"
$env:HYDRO_IYZICO_CALLBACK_URL="https://magaza.example.com/api/commerce/payments/iyzico/callback"

# PayTR
$env:HYDRO_PAYTR_MERCHANT_ID="merchant_id"
$env:HYDRO_PAYTR_MERCHANT_KEY="merchant_key"
$env:HYDRO_PAYTR_MERCHANT_SALT="merchant_salt"
```

---

## 3. Başlatma ve Yönetim

1. Migrasyonları çalıştırın:
   ```powershell
   hydro migrate --config config.yaml
   ```

2. Admin kullanıcısını oluşturun:
   ```powershell
   hydro create-admin --config config.yaml --email admin@magaza.test --password-stdin
   ```

3. Sunucuyu başlatın:
   ```powershell
   hydro serve --config config.yaml
   ```

Erişim noktaları:
- **Mağaza / Ürün Vitrini:** `http://127.0.0.1:8080/shop`
- **Sepet:** `http://127.0.0.1:8080/cart`
- **Admin Ürün & Stok Yönetimi:** `http://127.0.0.1:8080/admin/commerce/products`
- **REST API Ürünler:** `http://127.0.0.1:8080/api/v1/commerce/products`
