# HydroGo

Müşteri başına ayrı deployment ile çalışan, Core fork gerektirmeyen modüler Go CMS ve JSON API platformu.

**Durum:** R0, R1, R2 ve R4 tamamlandı; R3 ticari pilot ve R5 kararlı v1 doğrulama kapıları sürüyor. Pages/News modülleri, SQLite/PostgreSQL sözleşmeleri, backup/restore, güvenlik, generator ve CI uygulanmış durumda. Açık işler ve gerçek ortam kanıtları [roadmap](docs/ROADMAP.md) içinde izlenir.

- [Roadmap](docs/ROADMAP.md): iş sırası ve kabul ölçütleri.
- [Manifesto](docs/HYDROGO_PHILOSOPHY.md): ürün ilkeleri.
- [Uygulama planı](docs/implementation_plan.md): kapsam ve hedef mimari.
- [ADR indeksi](docs/architecture/README.md): bağlayıcı kararlar.
- [Hazırlık incelemesi](docs/PRE_IMPLEMENTATION_REVIEW.md) ve [tehdit modeli](docs/security/THREAT_MODEL.md): kanıt bekleyen riskler.
- [Katkı akışı](CONTRIBUTING.md) ve [depo kuralları](AGENTS.md).

## Kaynak yerleşimi

| Dizin | Sorumluluk |
|---|---|
| `cmd/hydro/` | Uygulama ve işletim alt komutlarının giriş noktası |
| `internal/` | Core uygulama ayrıntıları; dış SDK değildir |
| `extension/` | Deneysel v0 müşteri/modül sözleşmeleri |
| `modules/pages/`, `modules/news/` | Modüle ait model, servis, repository ve sunum |
| `examples/client-edition/` | Ayrı Go module olacak örnek; CI ayrıca dış tüketici fixture'ı kullanır |
| `migrations/` | Dialect SQL ve uyumluluk metadata'sı |
| `static/` | Sürümü ve checksum'ı sabitlenecek dağıtım varlıkları |
| `tests/` | Ortak contract, entegrasyon ve dış tüketici fixture'ları |
| `tools/`, `scripts/` | Sabitlenmiş geliştirme araçları ve PowerShell akışı |
| `docs/validation/`, `docs/runbooks/` | Gerçek deney raporları ve doğrulanacak işletim prosedürleri |

Boş dizinlerdeki `.gitkeep` dosyaları yalnız Git yer tutucusudur. İç paketler ihtiyaç oluşunca ayrılır. Go module yolu `github.com/ardaozyildirim/hydrogo` olarak ayrılmıştır; `go.mod` ve toolchain R1 deneyinde oluşturulur. Windows geliştirme ve Linux amd64 release ilk hedeflerdir.

Kaynak kodun müşteri teslimi ve yeniden dağıtım lisansı henüz seçilmedi; açık kaynak lisansı verilmiş sayılmaz.

## Geliştirme iskeletini çalıştırma

Go 1.27.0 ile:

```powershell
go build -o bin/hydro.exe ./cmd/hydro
$env:HYDRO_DATABASE_DSN = "hydro.sqlite"
./bin/hydro.exe migrate
./bin/hydro.exe serve
```

Ardından `/healthz` proses sağlığını, `/readyz` DB bağlantısı ve migration checksum uygunluğunu gösterir. Mevcut migration Pages transaction deneyi içindir; henüz üretim şeması veya satılabilir CMS değildir. PostgreSQL için `HYDRO_DATABASE_DIALECT=postgres` ve izole bir `HYDRO_DATABASE_DSN` kullanılır.
