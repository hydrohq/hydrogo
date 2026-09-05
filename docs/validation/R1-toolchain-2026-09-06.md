# R1 araç zinciri başlangıç kaydı

**Tarih:** 2026-09-06  
**Roadmap maddesi:** R1 — Go module/toolchain ve çalışan iskelet başlangıcı  
**Durum:** Kısmi kanıt; S01–S04 tamamlanmadı.

## Ortam

- Geliştirme işletim sistemi: Windows.
- Go: Go Programming Language 1.27.0, Winget paketi `GoLang.Go` ile kuruldu.
- Module path: `github.com/ardaozyildirim/hydrogo`.
- İlk bağımlılık: yalnız Go standard kütüphanesi.

## Beklenen davranış

`go test ./...`, `go vet ./...` ve `go build ./cmd/hydro` başarılı olmalı. Geliştirme CLI'si `version` ve minimal `serve` komutlarını sunmalı; `/healthz` ve `/readyz` açıkça kaydedilmeli, bilinmeyen rota 404 dönmeli.

## Kanıt

2026-09-06 tarihinde Windows üzerinde aşağıdaki kontroller başarılı oldu:

```text
go version
go test ./...
go vet ./...
go build -o bin/hydro.exe ./cmd/hydro
hydro version
```

`go test ./...` iki paketi doğruladı: CLI komut ayrıştırması ve yalnız açıkça kaydedilmiş operasyonel rotalar. CI workflow aynı format, vet, test ve build kontrollerini Linux üzerinde tekrarlar.

## Sınırlar

Bu iskelet, DB bağlantısı veya migration, config dosyası, session/bootstrap, CSRF/audit, Client Edition import sınırı ya da Templ/CSP kanıtı içermez. `/readyz` şu anda yalnız proses içi başlangıç hazır oluşunu temsil eder; DB uygunluğu R1'in DB deneyi tamamlanmadan ilan edilmez.
