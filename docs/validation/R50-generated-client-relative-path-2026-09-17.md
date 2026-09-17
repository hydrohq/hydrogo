# R50 — Generated client relative-path boundary

**Date:** 2026-09-17  
**Status:** Başarılı (yerel generator contract)

Generated `HydroClient` artık `mutation()` ve dahili request katmanında yalnızca
relative API path kabul ediyor. Mutlak veya protocol-relative bir URL için
fail-closed `RangeError` üretiliyor; böylece yapılandırılmış API anahtarı yanlış
bir hosta gönderilemiyor.

Doğrulama:

- `go test ./internal/projectgen`
- `scripts/verify-local.ps1`

Bu kontrol istemci tarafı bir güvenlik sınırıdır; sunucu tarafı yetkilendirme ve
gerçek production proxy/HTTPS kanıtlarının yerine geçmez.
