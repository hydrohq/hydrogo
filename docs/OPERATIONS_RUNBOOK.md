# HydroGo işletim runbook'u

Bu belge Compact pilot için uygulanabilir işletim adımlarını toplar. Restore ve üretim kapasite iddiaları, ilgili validation raporunda gerçek ortam kanıtı bulunmadan sertifikalı kabul edilmez.

## Deploy ve rollback

1. Yeni binary, config ve migration manifest'i staging dizinine kopyalanır; `hydro migrate` çalıştırılır.
2. Üretimde HTTPS `HYDRO_PUBLIC_ORIGIN` ile birlikte `HYDRO_SECURE_COOKIES=true` ve `HYDRO_ENABLE_HSTS=true` ayarlanır; bu üçlüden biri eksik veya çelişkiliyse config başlangıçta reddedilir. HTTPS sonlandırmasının gerçekten etkin olduğu doğrulanır. `hydro version --json` çıktısı deployment kaydına alınarak binary commit'i ve embedded schema migration seviyesi saklanır.
3. `hydro doctor` ile seçilen kapasite profili, DB bağlantısı/checksum'ları, backup araçları ve medya dizininin yazılabilirliği doğrulanır; ardından `hydro serve` başlatılır ve `/readyz` beklenir.
4. `/metrics` yalnızca authenticated operasyon erişimiyle okunur ve cache’lenmez; HTTP toplamı, in-flight, sınırlı status sınıfı, worker sonucu ve session cleanup sayaçları alarm/kapasite sistemine aktarılır.
5. Her release doğrulamasında `go run golang.org/x/vuln/cmd/govulncheck@v1.8.0 ./...` çalıştırılır; reachable vulnerability sonucu temiz değilse release durdurulur ve advisory değerlendirilir.
6. `hydro serve` yapılandırılmış JSON HTTP loglarını stderr'e yazar. Loglarda request ID, method, path, status ve süre bulunur; query string, gövde, cookie ve response body bulunmaz.
7. Blue/green geçişinde green slot trafik almadan önce public route, private media, login, sitemap ve form submit smoke istekleri çalıştırılır.
8. Eski slot yeni yazma almaya devam edemeyecek şekilde durdurulur; worker lease'leri drain/expiry politikasına bırakılır.
9. Hata durumunda binary rollback yalnız eski şema ve content/event/job formatlarıyla uyumluluk doğrulanırsa yapılır. Yedekten restore ayrı prosedürdür.

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

Snapshot retention için önce `hydro backup prune --root SNAPSHOTS --keep 3 --dry-run` çıktısı incelenir; yalnızca onaylanan adaylar `--confirm` ile silinir. Pinned snapshot adları `--pinned` ile belirtilir ve hiçbir koşulda silinmez. Manifest'i eksik veya bozuk snapshot ağacı fail-closed olarak işlemi durdurur.

## Alarm değerlendirmesi

`internal/ops.Evaluate` son başarılı backup yaşı, minimum disk alanı, migration başarısızlığı, SMTP delivery failure ve dead-letter job sinyallerini deterministic alarm kodlarına çevirir. `backup_stale`, `disk_low` ve `migration_failed` kritik; `smtp_delivery` ve `jobs_dead_lettered` uyarı seviyesindedir. Evaluator alarm delivery sisteminin yerine geçmez; production notifier ve dış backup/key kurtarma tatbikatı ayrıca doğrulanmalıdır.

News worker concurrency seçilen kapasite profilinden gelir: Compact tek worker, Standard dört, Distributed on altı bounded worker kimliği kullanır. Worker lease süresi ve queue backpressure ölçülmeden profil override edilmemelidir.

Dead-letter job'lar önce `hydro jobs dead-letter list` ile incelenir. Payload CLI çıktısına dahil edilmez. Operatör doğrulanmış bir düzeltmeden sonra yalnızca açık onayla `hydro jobs dead-letter requeue --id ID --confirm` çalıştırır; attempt ve son hata tanısı korunur.

Rate-limit yanıtları `429` ve `Retry-After` başlığı içerir; istemci ve proxy katmanı bu süre dolmadan yeniden deneme yapmamalıdır. Login/MFA ve form limiter'ları boş anahtarları reddeder; distributed limiter backend'e boş anahtar göndermez.

Artık gereksiz dead-letter kayıtları yalnızca saklama süresi ve batch sınırı verilerek temizlenir: `hydro jobs dead-letter purge --older-than 720h --limit 100 --confirm`. Aktif veya claim edilebilir işler bu işlemden etkilenmez.

## Mevcut limitler

- Login/MFA: process-local limiter, 10.000 anahtar, anahtar başına 5 deneme / 1 dakika.
- Contact form: 10.000 anahtar, form+IP başına 5 gönderim / 1 dakika; body en fazla 64 KiB.
- Media upload: 10 MiB.
- Public page cache: 256 kayıt, 30 saniye TTL.
- Session: 7 gün mutlak TTL, 24 saat idle TTL.
- Süresi dolan session'lar başlangıçta ve beş dakikalık periyodik worker ile temizlenir; `hydro_session_cleanup_runs_total`, `hydro_session_cleanup_rows_total` ve `hydro_session_cleanup_errors_total` metrikleri izlenir.

Bu değerler pilot trafiğiyle ölçülüp R3 kapasite raporunda güncellenmelidir; varsayılanlar donanım veya RPO/RTO garantisi değildir.

## Kişisel veri inbox talepleri

Form inbox erişim, export ve silme adımları [PERSONAL_DATA_POLICY.md](PERSONAL_DATA_POLICY.md) ile sınırlıdır. Operatör önce admin yetkisini ve talep kimliğini doğrular, sonra ilgili audit kaydını açar; export şifreli geçici konuma alınır ve teslim sonrası silinir. Silme işlemi dış erasure checkpoint'i ile replay edilmeden tamamlandı olarak işaretlenmez.
