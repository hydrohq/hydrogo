# R5 üretilen SDK rehberi doğrulaması — 2026-09-16

**Durum:** Başarılı (lokal/WSL)

`hydro project new` çıktısına `docs/SDK.md` eklendi. Belge; lockfile, lokal workflow, runtime dağıtım sınırı ve `custom/`, `themes/`, `migrations/` extension sınırlarını proje üretildiği anda açıklar.

## WSL kanıtı

```bash
cd /path/to/project
go test ./internal/projectgen ./cmd/hydro ./internal/generator ./internal/release ./internal/sdkdist
go run ./cmd/hydro project new sdk-doc --preset compact-site --no-demo --root /tmp/hydro-sdk-doc-check
test -f /tmp/hydro-sdk-doc-check/sdk-doc/docs/SDK.md
```

Gerçek sonuç: generator ve CLI testleri başarılı; `sdk-doc/docs/SDK.md` üretildi (`sdk-doc-ok`).

## Sınır

Bu belge runtime artifact broker veya installer uygulamaz; bu bileşenlerin henüz uygulanmadığını generated project içinde açıkça belirtir.

Güncel şablon ayrıca `hydro check`, `resource new/check`, `permissions check`,
`api diff` ve imzalı `schema diff` adımlarını gösterir. Template değişikliğinden
sonra Windows'ta `go test ./internal/projectgen` başarılıdır; WSL tam suite’i
ayrı cross-platform kanıt olarak tutulur.
