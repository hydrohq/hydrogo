# ADR-0002: SQLite, MySQL ve PostgreSQL desteği

- **Durum:** Değiştirildi — [ADR-0006](0006-extensible-database-adapters.md)
- **Tarih:** 2026-09-05

## Bağlam

HydroGo 512 MB RAM'li küçük VPS'ten yüksek trafikli ve çok düğümlü kurulumlara kadar çalışmalıdır. Tek veritabanı seçimi ya küçük kurulumları gereksiz ağırlaştırır ya da büyük kurulumları sınırlar.

## Karar

HydroGo üç birinci sınıf veritabanı profili sunar:

| Sürücü | Ana kullanım | Deployment sınırı |
|---|---|---|
| SQLite | Nano/Standalone, tek düğüm | Yerel kalıcı disk, tek aktif uygulama düğümü |
| PostgreSQL | Scale-out ve yüksek eşzamanlılık | Birden fazla HydroGo instance, managed veya ayrı DB |
| MySQL | MySQL standardı bulunan kurumlar | InnoDB, desteklenen sürüm matrisi |

`psql`, PostgreSQL'in komut satırı aracıdır; mimari sürücü adı `postgres` olacaktır.

SQLite ilk referans uygulamadır. PostgreSQL ve MySQL adaptörleri kararlı `v1` sürümünden önce aynı repository contract testlerinden geçmelidir.

## Tasarım kuralları

### Alan ve servis katmanı

- Domain ve application service kodu SQL dialect'i bilmez.
- Repository arayüzleri onları kullanan modül tarafından tanımlanır.
- Transaction sınırı application service tarafından istenir, platform tarafından sürücüye uygulanır.
- Veritabanına özgü hata kodları ortak `conflict`, `not_found`, `unavailable` ve `retryable` sınıflarına çevrilir.

### Fiziksel şema

- Her dialect normal tablo ve index kullanır; EAV ana veri modeli kullanılmaz.
- ID'ler uygulama tarafında üretilen taşınabilir UUID/ULID olur.
- Tarihler UTC tutulur.
- Para tamsayı minor-unit ve currency ile saklanır; floating point kullanılmaz.
- Boolean, JSON, full-text, generated column ve collation farkları capability olarak ele alınır.
- Core işlevi yalnız bir veritabanının özel özelliğine zorunlu biçimde bağlanmaz.

### Sorgular ve kod üretimi

Ortak davranış tek service sözleşmesinde kalır; SQL gerektiğinde dialect'e göre ayrı yazılır:

```text
modules/news/store/sqlite/
modules/news/store/postgres/
modules/news/store/mysql/
```

`sqlc` değerlendirme adayıdır; güncel dokümantasyon Go için MySQL ve PostgreSQL desteğini kararlı, SQLite desteğini beta olarak tanımlar. Bu nedenle seçim, üç sürücülü prototip ve benchmark tamamlanmadan kalıcı mimari kural değildir.

### Migration

Her mantıksal migration üç dialect uygulamasına sahip olabilir:

```text
migrations/202609050001_create_articles/
├── sqlite.sql
├── postgres.sql
├── mysql.sql
└── migration.yaml
```

`migration.yaml` şunları taşır:

- Migration kimliği ve checksum
- `online`, `backfill` veya `maintenance` sınıfı
- Geriye uyumlu binary aralığı
- Tahmini kilit ve disk etkisi
- Geri dönüş stratejisi

Migration bir kez yayımlandıktan sonra değiştirilmez.

## Test matrisi

Her modül için aynı davranış testleri şu matris üzerinde çalışır:

```text
SQLite × current
PostgreSQL × desteklenen ana sürümler
MySQL/InnoDB × desteklenen ana sürümler
```

Testler CRUD ile sınırlı değildir; optimistic locking, isolation, unique conflict, rollback, deadlock retry, pagination ve migration senaryolarını içerir. PostgreSQL ve MySQL farklı concurrency/locking davranışlarına sahip olduğundan yalnız aynı SQL'i çalıştırmak eşit destek sayılmaz.

## Yedekleme

- SQLite: Online Backup API veya doğrulanmış `VACUUM INTO`, immutable medya manifestiyle birlikte.
- PostgreSQL: Sağlayıcının snapshot/PITR sistemi veya PostgreSQL uyumlu araçlar.
- MySQL: Sağlayıcının snapshot/PITR sistemi veya MySQL uyumlu araçlar.

HydroGo her motorun kendi güvenilir yedekleme yöntemini orkestre eder; bütün motorlara tek dosya kopyalama yaklaşımı uygulamaz.

## Sonuçlar

- Küçük kurulumlar harici DB servisine ihtiyaç duymaz.
- Büyük kurulumlar yatay ölçeklenebilir.
- Üç dialect geliştirme ve CI maliyetini belirgin artırır.
- Yeni modül üç sürücü için tamamlanmadan “birinci sınıf” kabul edilmez.
- En düşük ortak payda uğruna performans feda edilmez; capability ve sürücüye özel optimize sorgular kullanılabilir.

## Kaynaklar

- [PostgreSQL concurrency control](https://www.postgresql.org/docs/current/mvcc.html)
- [MySQL InnoDB transaction model](https://dev.mysql.com/doc/refman/8.4/en/innodb-transaction-model.html)
- [sqlc database support](https://docs.sqlc.dev/en/latest/reference/language-support.html)
