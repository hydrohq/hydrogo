# Admin Studio liste doğrulaması — 2026-09-10

İlgili plan: [HEADLESS_ADMIN_STUDIO_PLAN.md](../HEADLESS_ADMIN_STUDIO_PLAN.md). Güncel seed edilmiş SQLite demo binary'si üzerinde ortak metadata list renderer'ı doğrulandı.

## Otomatik kanıt

- `go test ./...`
- `internal/adminstudio` form/list/registry testleri
- Pages service contract'ı: published summary listesi draft body ve blocks taşımaz.

## Yerel HTTP smoke

Demo credentials ile session açıldıktan sonra:

| Rota | Sonuç |
|---|---|
| `GET /admin/news` | `200`; ortak `<table>` ve `HydroGo Demo Release Notes` görünüyor |
| `GET /admin/pages` | `200`; ortak `<table>` ve `HydroGo Showcase` görünüyor |

Genişletilmiş güncel binary smoke'unda `GET /admin/commerce/products/demo-shirt/variants` de `200` ve ortak tablo/form renderer'ını kullandı. `GET /admin/pages/demo-showcase` blok editörünü, `GET /admin/news/demo-news` ise metadata textarea'sını ve schedule/publish workflow aksiyonlarını birlikte sundu.

Her satırın `Düzenle` aksiyonu modülün mevcut detay route'una gider. Mutation ve publish/schedule kuralları hâlâ ilgili application service'lerine aittir; liste renderer'ı DB erişimi veya yetki kararı taşımaz.
