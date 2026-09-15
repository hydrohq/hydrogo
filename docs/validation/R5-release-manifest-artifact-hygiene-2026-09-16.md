# R5 — Release manifest artifact hygiene

Runtime dağıtım zinciri, imzalı manifest doğrulamasında artifact yollarını
tekilleştirir ve her girdinin güvenli göreli path, negatif olmayan size ve
geçerli SHA-256 metadata'sı taşımasını zorunlu kılar. Duplicate path, absolute
path, traversal veya bozuk checksum kabul edilmez.

Windows doğrulaması:

```text
go test ./internal/release ./internal/sdkdist ./cmd/hydro
PASS
```

Bu kontrol imzayı doğrular; trusted key yönetimi ve dış release pipeline'ı yine
production kapısıdır.
