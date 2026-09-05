# S01 Client Edition Sınırı ve Sıfır Yük Doğrulaması

**Tarih:** 2026-09-06  
**Roadmap maddesi:** R1 / S01  
**Durum:** Başarılı.

## Doğrulanan Davranış

1. **Paket Sınırı ve İzolasyon:**
   - `examples/client-edition` bağımsız bir Go module olarak Core kaynağını kopyalamadan `extension` paketini import edip derlendi.
   - `tools/importguard`, Client Edition kaynaklarında HydroGo module yolundan yalnız ilan edilen `extension` prefix'ine izin verdi.
   - Ayrı `tests/fixtures/internal-import-denied` module'ü `internal/server` import etmeye çalıştığında Go 1.27.0 derleyicisi `use of internal package ... not allowed` ile build'i reddetti.

2. **Seçilmeyen Modülün Sıfır Yük Üretmesi:**
   - `extension.Module` ve `internal/modules.Registry` mekanizması ile hiçbir modül kaydedilmediğinde HTTP router'ında sıfır ek rota ve sıfır ek bellek/işlem yükü oluştuğu test edildi (`internal/modules/registry_test.go`).
   - Modüller seçilmedikçe ne route ne de middleware overhead'i üretir.

## Çalıştırılan Komutlar

```text
go test ./...
go vet ./...
go run ./tools/importguard -root examples/client-edition
cd examples/client-edition && go build ./...
```
Tüm kontroller başarıyla tamamlandı.
