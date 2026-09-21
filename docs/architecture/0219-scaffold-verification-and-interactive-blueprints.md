# ADR-0219: Scaffold doğrulama ve interaktif Blueprint akışı

## Durum

Kabul edildi — 2026-09-22.

## Karar

`hydro generate scaffold` varsayılan olarak yalnızca üretilen Go dosyalarını
formatlar ve mevcut bir `go.mod` varsa bounded `go test ./...` doğrulaması
yapar. `go mod tidy`, `templ generate` ve benzeri çalışma ağacını genişleten
araçlar otomatik çalıştırılmaz. `hydro new resource` aynı Blueprint formatını
interaktif sorularla üretir; ikinci bir model veya kod üretim yolu yoktur.

## Sonuçlar

Üretilen kod gözden geçirilebilir ve derlenebilir kalır; hook'lar müşteri
modülünün bağımlılıklarını veya migration'larını sessizce değiştirmez.
