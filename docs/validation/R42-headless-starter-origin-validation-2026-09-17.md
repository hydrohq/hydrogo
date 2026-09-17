# R42 — Headless starter origin doğrulaması

**Date:** 2026-09-17  
**Status:** Başarılı (yerel Node runtime)

Headless client constructor artık absolute HTTP(S) origin ister ve embedded
credentials ile `javascript:`/diğer şemaları fail-closed reddeder. Böylece
yanlış yapılandırılmış veya kopyalanmış bir URL, fetch katmanına ulaşmadan
görünür bir `RangeError` üretir.

```powershell
Push-Location examples/headless-starter
npm test
Pop-Location
```

Sonuç: **PASS** — origin scheme/credential sınırları, external abort, timeout,
idempotency, typed error/request-id ve bounded response kontrolleri.
