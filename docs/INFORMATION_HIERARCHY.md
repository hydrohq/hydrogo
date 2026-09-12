# HydroGo Bilgi Hiyerarsisi ve Kullaniciya Saygi Ilkeleri (Information Hierarchy & Audience Separation)

> **"Kullaniciya saygi, sadece hizli yanit vermek degil; onu anlamadigi, ihtiyaç duymadigi ve bilissel yük olusturan teknik gürültüden korumaktir."**

HydroGo platformunda sunulan her türlü arayüz, mesaj, log ve API yaniti belirli bir hedef kitle (audience) gözetilerek tasarlanir. Hiçbir gereksiz teknik bilgi ve iç mimari detay son kullaniciya (müsteriye) sizdirilamaz.

---

## ?? 5 Temel Bilgi Katmani (Information Tiers)

Platform genelinde bilgi 5 farkli seviyede ayristirilir ve filtrelenir:

```
+-------------------------------------------------------------+
¦ 1. MÜSTERI / SON KULLANICI (Member & Public Customer)       ¦  ? Sade, nazik, eyleme dönük, sifir teknik jargon
+-------------------------------------------------------------¦
¦ 2. ISLETME YÖNETICISI / EDITÖR (Admin Studio User)          ¦  ? Içerik, siparis, katalog, müsteri ve is metrikleri
+-------------------------------------------------------------¦
¦ 3. SISTEM YÖNETICISI / OPERATÖR (Sysadmin & SRE)             ¦  ? Sunucu sagligi, DB havuzu, backup, drift, kuyruklar
+-------------------------------------------------------------¦
¦ 4. GELISTIRICI & ENTEGRATÖR (Developer & API Consumer)      ¦  ? API semalari, webhook loglari, payloadlar, SDK'ler
+-------------------------------------------------------------¦
¦ 5. DENETÇI & UYUM GÖREVLISI (Auditor & Compliance / DPO)    ¦  ? Denetim izleri (audit logs), veri silme, lisans kaniti
+-------------------------------------------------------------+
```

---

### 1. Müsteri / Son Kullanici Katmani (Member & Public Customer)

**Amaç:** Kullanicinin isini en az çaba ve maksimum güvenle halletmesini saglamak.  
**Dil ve Ton:** Net, nazik, anlasilir, çözüm odakli Türkçe / Ingilizce.

- ? **GÖSTERILECEKLER:**
  - Siparis durumu, kargo/lisans kodu, fatura indirme baglantilari.
  - Hesap bilgileri, kayit tarihi, üyelik rozeti.
  - Basit ve anlasilir güvenlik durumu ("Güvenli", "2 Cihazda Açik", "Son Giris: Bugün 14:30").
  - Anlasilir hata mesajlari ("E-posta veya sifre hatali", "Bu e-posta adresi zaten kayitli", "Lütfen bilgilerinizi kontrol edin").
  - Açik yönlendirme ve eylem butonlari ("Sepete Git", "Güvenli Çikis", "Sifremi Sifirla").

- ? **KESINLIKLE GÖSTERILMEYECEKLER (Yasak Bilgiler):**
  - Kriptografik algoritma ve kütüphane isimleri (Örnek: `Argon2id`, `AES-256-GCM`, `Ed25519`, `bcrypt`, `crypto/subtle`).
  - Veritabani ve motor ayrintilari (Örnek: `SQLite`, `PostgreSQL`, `pgx`, `database/sql`, SQL sorgulari, tablo/sütun adlari).
  - Sunucu/Kod istisnalari ve hata yiginlari (Örnek: Go panic, stack trace, `nil pointer dereference`, file paths `C:\dev\...`).
  - Dahili sistem kimlikleri ve hash dizgileri (Örnek: session token hash'leri, salt degerleri, internal worker ID'ler).
  - Ham HTTP / Altyapi metrikleri (Örnek: response time ms, connection pool occupancy, memory allocation).

---

### 2. Isletme Yöneticisi / Editör Katmani (Admin Studio User)

**Amaç:** Günlük ticari ve editoryal operasyonlari kolayca yürütmek.  
**Erisim Alani:** `/admin`, `/admin/news`, `/admin/pages`, `/admin/commerce/*`, `/admin/members`.

- ? **GÖSTERILECEKLER:**
  - Haber taslaklari, yayinlanma zamanlari, kategori ve etiketler.
  - Ürün stoklari, varyant fiyatlari, siparis tutarlari ve müsteri iletisim detaylari.
  - Satis grafikleri, sepet dönüsüm oranlari, son kayit olan üyeler.
  - Rich Text editör araçlari, medya yükleme ve görsel önizlemeleri.

- ? **GÖSTERILMEYECEKLER:**
  - Sunucu CPU/RAM ham donanim sayaçlari.
  - Outbox kuyruk retry gecikme algoritmalari veya ham SQL migration diff'leri.
  - Sysadmin seviyesindeki yedekleme restore fence kilitleri.

---

### 3. Sistem Yöneticisi / Operatör Katmani (Sysadmin & SRE)

**Amaç:** Sistemin yüksek erisilebilirligini, sifir kesintili dagitimlari ve felaket kurtarma süreçlerini garanti etmek.  
**Erisim Alani:** CLI (`hydro doctor`, `hydro backup`, `hydro release`), `/admin/control`, `/admin/operations`.

- ? **GÖSTERILECEKLER:**
  - Veritabani baglanti havuzu (active / idle connections, max pool limit).
  - Blue/Green dagitim durumu, write fence ve rollback hazir bulunuslugu.
  - Disk doluluk orani, WAL boyutu, yedekleme bütünlük hash'leri (SHA-256).
  - Dead-letter kuyruk dolulugu, outbox retry sayaçlari ve background worker sagligi.
  - Rate limiting sayaçlari, bellek tüketimi ve lisans geçerlilik süresi (grace period takibi).

---

### 4. Gelistirici & Entegratör Katmani (Developer & API Consumer)

**Amaç:** Harici sistemlerle API, webhook ve modül entegrasyonu saglamak.  
**Erisim Alani:** `/admin/developer`, `/api/v1/*`, GraphQL semasi, Webhook loglari.

- ? **GÖSTERILECEKLER:**
  - REST ve GraphQL OpenAPI sema tanimlari.
  - Webhook teslim durumu, HTTP durum kodlari (200, 429, 500) ve giden/gelen JSON payloadlari.
  - API token'lari (maskelenmis), yetki kapsamlari (scopes) ve rate limit kotalari.
  - Tip güvenli Go SDK ve Client Edition kontrat belgeleri.

---

### 5. Denetçi & Uyum Görevlisi Katmani (Auditor & Compliance / DPO)

**Amaç:** Hukuki yükümlülükleri, KVKK/GDPR uyumlulugunu ve güvenlik standartlarini denetlemek.  
**Erisim Alani:** `/admin/security`, `/admin/permissions`, Audit Trail loglari.

- ? **GÖSTERILECEKLER:**
  - Degistirilemez (append-only) denetim günlükleri: Kim, ne zaman, hangi kaydi degistirdi/sildi.
  - KVKK/GDPR kisisel veri silme ve anonimlestirme sertifikalari/tutanaklari.
  - Güvenlik ihlali / basarisiz oturum açma denemeleri ve süpheli IP kaliplari.
  - Ed25519 lisans imza dogrulama sertifikasi ve yasal kullanim sinirlari.

---

## ??? Yazilim Gelistirme ve Tasarim Kurallari

1. **Fail-Closed & Bilgi Sizdirmazlik:** Hata durumlarinda kullaniciya asla "internal error: sql syntax error at line 42" verilmez. Kullaniciya "Islem su anda gerçeklestirilemiyor, lütfen daha sonra tekrar deneyin" denirken, detay sysadmin loguna (`logger.Error`) yazilir.
2. **Kelimelerin Seçimi:** Kriptografi ve mimari terimleri sadece yetkili teknik personelin ekranlarinda yer alir. Müsteri ekraninda "Argon2id" yerine "Sifreli Güvenlik", "HTTP 401 Unauthorized" yerine "Lütfen giris yapin" denir.
3. **Bilissel Yükü Sifirlama:** Bir ekrana yeni bir alan veya metin eklenmeden önce "Bu bilgi son kullanicinin bir karar almasina yardimci oluyor mu?" sorusu sorulmalidir. Yanit "Hayir" ise o bilgi son kullanici arayüzünden çikarilmalidir.
