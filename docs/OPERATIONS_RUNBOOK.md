# HydroGo işletim runbook'u

Bu belge Compact pilot için uygulanabilir işletim adımlarını toplar. Restore ve üretim kapasite iddiaları, ilgili validation raporunda gerçek ortam kanıtı bulunmadan sertifikalı kabul edilmez.

## Deploy ve rollback

1. Yeni binary, config ve migration manifest'i staging dizinine kopyalanır; `hydro migrate` çalıştırılır.
2. `hydro doctor` ile seçilen kapasite profili, DB bağlantısı/checksum'ları ve medya dizini doğrulanır; ardından `hydro serve` başlatılır ve `/readyz` beklenir.
3. `/metrics` yalnızca authenticated operasyon erişimiyle okunur; HTTP toplamı, in-flight ve sınırlı status sınıfı sayaçları alarm/kapasite sistemine aktarılır.
4. Blue/green geçişinde green slot trafik almadan önce public route, private media, login, sitemap ve form submit smoke istekleri çalıştırılır.
5. Eski slot yeni yazma almaya devam edemeyecek şekilde durdurulur; worker lease'leri drain/expiry politikasına bırakılır.
6. Hata durumunda binary rollback yalnız eski şema ve content/event/job formatlarıyla uyumluluk doğrulanırsa yapılır. Yedekten restore ayrı prosedürdür.

## Compact snapshot ve restore

Yazmalar ve worker dış egress kontrollü olarak durdurulur. SQLite için `hydro backup sqlite`, PostgreSQL için `hydro backup postgres` kullanılır; SQLite `VACUUM INTO`, PostgreSQL `pg_dump` custom format kullanır. Her iki akışta medya dosyaları checksum manifestiyle birlikte kopyalanır. PostgreSQL parolası geçici `PGPASSFILE` ile aktarılır; komut satırına yazılmaz.

Restore hedefi boş staging dizini olmalıdır. `Verify` geçmeden hiçbir dosya aktive edilmez. `RestoreToStaging` yalnız şu üç koşul kanıtlandığında marker üretir:

- Eski session kayıtları iptal edildi.
- Worker ve dış egress karantinaya alındı.
- Eski deployment'ın yazması engellendi.

Marker olmadan staging public trafiğe, worker'a veya export işlemine açılmaz. SQLite restore için `hydro restore stage`, PostgreSQL restore için boş hedef DB/instance üzerinde `hydro restore postgres` kullanılır; dosya dizini rename etmek PostgreSQL restore değildir. `hydro restore verify` aktivasyondan önce checksum kontrolünü bağımsız çalıştırır.

## Parola kurtarma ve davet

Reset ve invite token'ları yalnız hash olarak saklanır. Token e-posta/notifier adapter'ına verildikten sonra log veya audit'e yazılmaz. Bilinmeyen reset e-postası `202 Accepted` ile aynı cevabı alır. Invite kabulünde token tüketimi ve kullanıcı oluşturma tek transaction'dadır.

Üretimde notifier için `HYDRO_SMTP_HOST`, `HYDRO_SMTP_PORT`, `HYDRO_SMTP_FROM`, `HYDRO_SMTP_USERNAME`, `HYDRO_SMTP_PASSWORD` ve `HYDRO_SMTP_STARTTLS` secret/config kaynakları birlikte doğrulanır. SMTP teslimi başarısızsa reset/invite işlemi başarı ilan edilmez; provider retry ve alarm politikası deployment'a aittir.

## Disk dolması

Yeni upload ve snapshot işlemleri güvenli hata ile durdurulur; mevcut ready medya ve DB silinmez. Önce geçici/pending medya, başarısız snapshot ve süresi dolmuş inbox kayıtları retention politikasına göre temizlenir. Dosya sistemi tekrar yazılabilir olmadan migration, restore aktivasyonu veya GC çalıştırılmaz.

## Alarm değerlendirmesi

`internal/ops.Evaluate` son başarılı backup yaşı, minimum disk alanı, migration başarısızlığı ve SMTP delivery failure sinyallerini deterministic alarm kodlarına çevirir. `backup_stale`, `disk_low` ve `migration_failed` kritik; `smtp_delivery` uyarı seviyesindedir. Evaluator alarm delivery sisteminin yerine geçmez; production notifier ve dış backup/key kurtarma tatbikatı ayrıca doğrulanmalıdır.

## Mevcut limitler

- Login/MFA: process-local limiter, 10.000 anahtar, anahtar başına 5 deneme / 1 dakika.
- Contact form: 10.000 anahtar, form+IP başına 5 gönderim / 1 dakika; body en fazla 64 KiB.
- Media upload: 10 MiB.
- Public page cache: 256 kayıt, 30 saniye TTL.
- Session: 7 gün mutlak TTL, 24 saat idle TTL.

Bu değerler pilot trafiğiyle ölçülüp R3 kapasite raporunda güncellenmelidir; varsayılanlar donanım veya RPO/RTO garantisi değildir.

## Kişisel veri inbox talepleri

Form inbox erişim, export ve silme adımları [PERSONAL_DATA_POLICY.md](PERSONAL_DATA_POLICY.md) ile sınırlıdır. Operatör önce admin yetkisini ve talep kimliğini doğrular, sonra ilgili audit kaydını açar; export şifreli geçici konuma alınır ve teslim sonrası silinir. Silme işlemi dış erasure checkpoint'i ile replay edilmeden tamamlandı olarak işaretlenmez.
