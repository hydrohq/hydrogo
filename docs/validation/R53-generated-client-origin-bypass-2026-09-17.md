# R53 — Generated client origin-bypass hardening

**Date:** 2026-09-17  
**Status:** Başarılı (yerel generator contract)

`HydroClient` path sınırı backslash-normalizasyonlu URL bypass'larını da
fail-closed engelliyor. URL parser'ın `"/\\\\evil.example"` gibi bir girdiyi
başka bir origin'e çevirebilmesi nedeniyle yalnızca slash prefix kontrolü yeterli
değildi; artık çözümlenen URL origin'i yapılandırılmış `baseUrl` origin'iyle
karşılaştırılıyor.

Doğrulama:

- `go test ./internal/projectgen`
- `scripts/verify-local.ps1`

Bu kontrol generated istemcinin credential exfiltration riskini azaltır; sunucu
yetkilendirmesi ve production proxy politikalarının yerine geçmez.
