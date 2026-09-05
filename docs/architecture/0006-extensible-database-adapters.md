# ADR-0006: İlk sürüm SQLite/PostgreSQL ve genişletilebilir DB adaptörleri

- **Durum:** Kabul edildi
- **Tarih:** 2026-09-05
- **Değiştirdiği ADR:** [ADR-0002](0002-database-portability.md)

## Bağlam

Her veritabanını ilk sürümde eşit desteklemek geliştirme ve test maliyetini büyütür. Yalnız tek veritabanına göre tasarlamak ise ileride yeni motor eklemeyi zorlaştırır. HydroGo başlangıçta küçük kurulum ile çok instance'lı üretim kurulumunu kapsamalı, yeni veritabanlarını gerçek talep oluştuğunda ekleyebilmelidir.

## Karar

İlk kararlı sürümün sertifikalı veritabanları:

- **SQLite:** Compact, tek düğümlü ve kolay taşınabilir kurulum.
- **PostgreSQL:** Standard ve Distributed üretim kurulumları.

MySQL ve diğer SQL motorları başlangıç kapsamı değildir. Mimari bunların sonradan **sertifikalı adaptör** olarak eklenmesini destekler. “Bütün veritabanlarını destekler” vaadi verilmez.

## Sınırlar

### Platform adaptörü

Platform seviyesindeki adaptör yalnız altyapı davranışını sağlar:

```go
type Adapter interface {
    Name() string
    Capabilities() CapabilitySet
    Open(context.Context, Config) (Pool, error)
    Begin(context.Context, TxOptions) (Tx, error)
    ClassifyError(error) ErrorKind
}
```

Bu arayüz bütün SQL özelliklerini ortaklaştırmaya çalışmaz.

### Modül store'u

Her modül kullandığı repository/store sözleşmesini kendisi tanımlar. SQLite ve PostgreSQL sorguları ayrı uygulanır:

```text
modules/news/store/
├── contract.go
├── sqlite/
└── postgres/
```

Yeni bir veritabanı adaptörü eklemek yalnız connection driver yazmak değildir. Desteklenen her Core bileşeni ve modül için şunlar gerekir:

- Dialect sorguları
- Migration'lar
- Error mapping
- Transaction ve locking davranışı
- Backup/restore entegrasyonu
- Contract ve migration testleri

### Capability modeli

Adaptör özelliklerini açıkça bildirir:

```text
returning
json_native
full_text
skip_locked
advisory_lock
online_ddl
listen_notify
```

Modül gerekli capability'leri bildirir. Seçili veritabanı bunları sağlamıyorsa sistem başlangıçta açık hata verir veya modül kendi portable fallback'ini kullanır.

## Taşınabilir veri kuralları

- Kimlik uygulama tarafında üretilen UUID/ULID olur.
- Tarihler UTC ve açık hassasiyet sözleşmesiyle saklanır.
- Slug, e-posta ve benzersiz anahtarlar normalize edilmiş yardımcı alanlarla korunur; collation varsayımına bırakılmaz.
- Para tamsayı minor-unit ve currency koduyla saklanır.
- Pagination kararlı sıralama ve cursor sözleşmesi kullanır.
- Domain davranışı DB'ye özel trigger veya stored procedure'e zorunlu olarak bağlanmaz.

## Migration

Her migration seçili ve sertifikalı dialect uygulamalarına sahiptir:

```text
migrations/<id>/
├── sqlite.sql
├── postgres.sql
└── migration.yaml
```

İleride MySQL adaptörü eklenirse aynı mantıksal migration dizilerine `mysql.sql` eklenir veya motorun başlangıç baseline şeması üretilir. Eski migration checksum'ları değiştirilmez.

## Geliştirme ve test

- Hızlı yerel geliştirme SQLite ile yapılabilir.
- Müşterinin üretim hedefi PostgreSQL ise CI ve staging PostgreSQL kullanır.
- Her modül SQLite ve PostgreSQL ortak contract testlerini geçmeden tamamlanmış sayılmaz.
- Yeni adaptör aynı test kitini geçmeden “destekleniyor” olarak listelenmez.

## Sonuçlar

- İlk sürümün test matrisi yönetilebilir kalır.
- Küçük ve dağıtık iki gerçek topoloji doğrulanır.
- Yeni DB ekleme yolu açıktır fakat sahte evrensellik iddiası oluşmaz.
- DB'ye özel optimizasyon capability sınırları içinde kullanılabilir.
- Yeni DB desteği yine önemli bir ürün yatırımıdır; adaptör arayüzü bunu sıfır maliyetli yapmaz.
