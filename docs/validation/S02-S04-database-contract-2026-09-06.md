# S02/S04 veritabanı transaction ve migration doğrulaması

**Tarih:** 2026-09-06  
**Roadmap maddesi:** R1 / S02–S04 DB bölümü  
**Durum:** Başarılı; backup/encoder ve tam destek matrisi ayrı R1/P1 kapılarında bekliyor.

## Seçimler

- SQLite: `modernc.org/sqlite` v1.58.0. Saf Go sürücüsü Windows amd64 ve Linux amd64 hedeflerini kapsıyor; cgo toolchain gerektirmiyor. Sürücünün bağlı `modernc.org/libc` sürümü v1.75.6 olarak module graph içinde sabitlendi.
- PostgreSQL: `github.com/jackc/pgx/v5` v5.10.0, `database/sql` uyumluluk katmanı üzerinden.
- Migration: İlk deneyde harici migration framework'ü eklenmedi. Dialect SQL, değişmez SHA-256 metadata'sı ve transaction içindeki ledger davranışı küçük HydroGo runner'ıyla kanıtlandı. Daha geniş bir aracın gerekliliği sonraki migration örnekleriyle değerlendirilecek.

Kaynaklar: [modernc SQLite paket belgeleri](https://pkg.go.dev/modernc.org/sqlite), [pgx stdlib paket belgeleri](https://pkg.go.dev/github.com/jackc/pgx/v5/stdlib).

## Ortam ve sonuç

| Hedef | Veritabanı | Sonuç |
|---|---|---|
| Windows amd64, Go 1.27.0 | Dosya tabanlı SQLite / WAL | Başarılı |
| GitHub Actions Ubuntu, Go 1.27.0 | PostgreSQL 18 izole service container | Başarılı |

CI kanıtı: [run 33993049815](https://github.com/ardaozyildirim/hydrogo/actions/runs/33993049815).

## Kanıtlanan sözleşme

- SQLite her bağlantıda `foreign_keys=on`, `journal_mode=WAL`, `synchronous=NORMAL` ve `busy_timeout=5000` değerlerini uyguladı. Windows sürücü harfi için `file:///C:/...` URI düzeltmesi test sırasında bulundu ve kalıcılaştırıldı.
- Pages draft, audit olayı ve outbox olayı aynı `sql.Tx` executor'ını kullandı. Outbox'a enjekte edilen hata sonrası üç tabloda da satır kalmadı.
- Başarılı çağrıda üç kayıt birlikte commit edildi.
- Aynı page/version üzerinde eşzamanlı iki güncellemeden yalnız biri satır değiştirdi; sessiz son-yazan-kazan oluşmadı.
- Aynı migration'ın ikinci çalışması no-op oldu. SQL içeriği değiştirilince ledger checksum uyuşmazlığı işlemi reddetti.
- Migration metadata'sındaki iki dialect checksum'ı CI testiyle gerçek SQL dosyalarına bağlandı.

## Sınırlar

Bu deney üretim şeması değildir. PostgreSQL testi müşteri DB'sine değil, her CI çalışmasında oluşturulan `hydro_contract` veritabanına yazdı. Yerel PostgreSQL servisi parola verilmediği için kullanılmadı. Deadlock/serialization retry, online DDL, backup/restore, connection pool kapasitesi ve desteklenen PostgreSQL minor sürüm matrisi henüz sertifikalanmadı.
