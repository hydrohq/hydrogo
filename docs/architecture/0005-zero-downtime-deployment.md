# ADR-0005: Blue/green ve kesintisiz sürüm geçişi

- **Durum:** Değiştirildi (kısmen)
- **Tarih:** 2026-09-05
- **Yerine geçen hükümler:** [ADR-0011](0011-consistency-upgrade-and-recovery.md)

> Blue/green, proses rolleri, proxy ve expand–migrate–contract yaklaşımı geçerlidir. Warm-up/invalidation, üretimde contract testi, schema/job/content uyumluluğu ve restore ayrıntılarında ADR-0011 esas alınır. Kesintisizlik ürün hedefidir; sertifikalı topoloji ve test edilmiş release koşulları sağlanmadan gerçekleştirilmiş garanti değildir.

## Bağlam

HydroGo müşterileri içerik yayını sırasında kesinti yaşamamalı; aynı zamanda veritabanı migration'ları ve background job'lar veri bütünlüğünü bozmamalıdır.

## Karar

HydroGo public web trafiğinde blue/green deployment destekler. Kesintisiz garanti yalnız eski ve yeni binary ile geriye uyumlu `online` migration'lar için verilir. Geriye uyumsuz tablo dönüşümleri kontrollü bakım penceresi gerektirir.

## Proses rolleri

Aynı binary farklı rollerle çalışabilir:

```text
hydro serve --role=all
hydro serve --role=web
hydro serve --role=worker
```

- Compact: `all`
- Blue/green: İki web slotu ve tek aktif worker
- Distributed: Birden fazla web ve lease kullanan birden fazla worker

## Tek sunucu blue/green akışı

1. Yeni release pasif slota indirilir ve imzası doğrulanır.
2. `hydro upgrade check` config, disk, DB capability ve migration sınıfını doğrular.
3. Deployment öncesi yedek/snapshot alınır.
4. `online` expand migration uygulanır; migration lock çift çalışmayı önler.
5. Green web, worker kapalı ve farklı port/socket üzerinde başlar.
6. Readiness, DB contract ve dahili smoke test çalışır.
7. Kritik public yollar çağrılarak L1 cache ısıtılır.
8. Reverse proxy trafiği Green'e atomik olarak geçirir.
9. Blue yeni istek almayı bırakır ve mevcut isteklerini tamamlar.
10. Worker yeni release ile yeniden başlar veya liderlik Green'e geçer.
11. Blue önceki release olarak rollback süresi boyunca saklanır.

Caddy config reload yeni konfigürasyonu eskisini durdurmadan yükleyebildiği için varsayılan tek sunucu reverse proxy çözümüdür. Admin API yalnız yerel socket/arayüz üzerinden korunur.

## Migration sınıfları

### Online

Eski ve yeni binary aynı şemada birlikte çalışabilir. Genellikle yeni tablo, nullable kolon veya geriye uyumlu index/constraint hazırlığıdır.

### Backfill

Şema genişletilir; veri küçük ve tekrar başlanabilir parçalarla taşınır. Kod geçiş süresinde eski ve yeni gösterimi okuyabilir/yazabilir.

### Maintenance

Tablo rewrite, geriye uyumsuz constraint, kolon kaldırma veya eski binary'nin okuyamayacağı dönüşümdür. Deployment bakım sayfası ve açık RTO ile yürütülür.

## Expand–migrate–contract

Kırılabilir değişiklikler sürümlere yayılır:

1. **Expand:** Yeni yapı eklenir, eski yapı korunur.
2. **Migrate:** Veri arka planda taşınır ve doğrulanır.
3. **Switch:** Yeni kod yeni yapıyı asıl kaynak olarak kullanır.
4. **Contract:** Eski yapı ancak destek penceresi bittikten sonra kaldırılır.

## Veritabanı profilleri

- SQLite: Blue ve Green aynı makinede ve yerel diskte kısa süre birlikte çalışabilir; aynı anda tek writer sınırı korunur.
- PostgreSQL: Çoklu web instance ve rolling/canary deployment desteklenebilir.
- Gelecekte sertifikalanan adaptörler online migration ve çoklu instance contract testlerini ayrıca geçmelidir.
- Her sürücü için online DDL gerçekte doğrulanır; yalnız SQL sözdizimine bakılarak “online” etiketi verilmez.

## Session, medya ve asset

- Session proses belleğinde tek doğruluk kaydı olmaz; DB'de tutulduğu için slot değişiminde kullanıcı çıkış yapmaz.
- Medya immutable ve iki slot tarafından erişilebilir depoda bulunur.
- Asset URL'leri içerik hash'i taşır.
- Yeni release eski cache'li HTML'nin referans verdiği asset'leri cache ömrü boyunca sunar.

## Cache tutarlılığı

- Atomik slot geçişinde Green cache önceden ısıtılır.
- Kısa Blue/Green örtüşmesinde versioned cache key kullanılır.
- Canary veya uzun süre eşzamanlı trafik varsa L2 distributed cache/invalidation zorunludur.

## Rollback

Proxy önceki slota ancak mevcut şema önceki binary'nin ilan ettiği uyumluluk aralığındaysa döner. Migration geriye uyumsuzsa otomatik binary rollback yapılmaz; kontrollü restore veya ileri düzeltme uygulanır.

## Garanti

HydroGo şu ifadeyi kullanır:

> Geriye uyumlu sürümler public trafikte kesintisiz yayımlanır. Yapısal veri dönüşümleri güvenli ve önceden bildirilen bakım penceresiyle uygulanır.

## Kaynak

- [Caddy zero-downtime configuration reload](https://caddyserver.com/docs/getting-started#reloading-config)
