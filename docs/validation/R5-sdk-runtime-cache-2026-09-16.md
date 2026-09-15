# R5 SDK runtime cache sözleşmesi — 2026-09-16

**Durum:** Başarılı (lokal/WSL)

`internal/sdkdist.CachePath`, runtime artifact cache yolunu sürüm/ürün/OS/mimariye göre deterministik üretir. Sürüm ve ürün girdileri path traversal’dan arındırılır; fonksiyon dizin oluşturmaz veya binary çalıştırmaz.

## Kanıt

- Windows: `go test ./internal/sdkdist` başarılı.
- WSL Ubuntu/Go 1.27: `go test ./internal/sdkdist` ve `go test ./...` başarılı.
- `../`, `..`, boş ve slash içeren sürüm değerleri reddedildi.

## Sınır

Bu yalnızca cache path sözleşmesidir. Download, atomic install, cache eviction, version switching ve broker entegrasyonu sonraki fazlardır.
