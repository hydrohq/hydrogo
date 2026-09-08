# HydroGo işletim runbook'u

Bu belge Compact pilot için uygulanabilir işletim adımlarını toplar. Restore ve üretim kapasite iddiaları, ilgili validation raporunda gerçek ortam kanıtı bulunmadan sertifikalı kabul edilmez.

## Deploy ve rollback

1. Yeni binary, config ve migration manifest'i staging dizinine kopyalanır; `hydro migrate` çalıştırılır.
2. `hydro doctor` ile seçilen kapasite profili, DB bağlantısı/checksum'ları ve medya dizini doğrulanır; ardından `hydro serve` başlatılır ve `/readyz` beklenir.
3. Blue/green geçişinde green slot trafik almadan önce public route, private media, login, sitemap ve form submit smoke istekleri çalıştırılır.
4. Eski slot yeni yazma almaya devam edemeyecek şekilde durdurulur; worker lease'leri drain/expiry politikasına bırakılır.
5. Hata durumunda binary rollback yalnız eski şema ve content/event/job formatlarıyla uyumluluk doğrulanırsa yapılır. Yedekten restore ayrı prosedürdür.

## Compact snapshot ve restore

Yazmalar ve worker dış egress kontrollü olarak durdurulur. `internal/backup.SnapshotSQLite` ile boş bir snapshot dizinine SQLite `VACUUM INTO` alınır; medya dosyaları checksum manifestiyle birlikte kopyalanır.

Restore hedefi boş staging dizini olmalıdır. `Verify` geçmeden hiçbir dosya aktive edilmez. `RestoreToStaging` yalnız şu üç koşul kanıtlandığında marker üretir:

- Eski session kayıtları iptal edildi.
- Worker ve dış egress karantinaya alındı.
- Eski deployment'ın yazması engellendi.

Marker olmadan staging public trafiğe, worker'a veya export işlemine açılmaz. PostgreSQL restore ayrı boş DB/instance'a `pg_restore` ile yapılır; dosya dizini rename etmek PostgreSQL restore değildir.

## Parola kurtarma ve davet

Reset ve invite token'ları yalnız hash olarak saklanır. Token e-posta/notifier adapter'ına verildikten sonra log veya audit'e yazılmaz. Bilinmeyen reset e-postası `202 Accepted` ile aynı cevabı alır. Invite kabulünde token tüketimi ve kullanıcı oluşturma tek transaction'dadır.

## Disk dolması

Yeni upload ve snapshot işlemleri güvenli hata ile durdurulur; mevcut ready medya ve DB silinmez. Önce geçici/pending medya, başarısız snapshot ve süresi dolmuş inbox kayıtları retention politikasına göre temizlenir. Dosya sistemi tekrar yazılabilir olmadan migration, restore aktivasyonu veya GC çalıştırılmaz.

## Mevcut limitler

- Login/MFA: process-local limiter, 10.000 anahtar, anahtar başına 5 deneme / 1 dakika.
- Contact form: 10.000 anahtar, form+IP başına 5 gönderim / 1 dakika; body en fazla 64 KiB.
- Media upload: 10 MiB.
- Public page cache: 256 kayıt, 30 saniye TTL.
- Session: 7 gün mutlak TTL, 24 saat idle TTL.

Bu değerler pilot trafiğiyle ölçülüp R3 kapasite raporunda güncellenmelidir; varsayılanlar donanım veya RPO/RTO garantisi değildir.

## Kişisel veri inbox talepleri

Form inbox erişim, export ve silme adımları [PERSONAL_DATA_POLICY.md](PERSONAL_DATA_POLICY.md) ile sınırlıdır. Operatör önce admin yetkisini ve talep kimliğini doğrular, sonra ilgili audit kaydını açar; export şifreli geçici konuma alınır ve teslim sonrası silinir. Silme işlemi dış erasure checkpoint'i ile replay edilmeden tamamlandı olarak işaretlenmez.
