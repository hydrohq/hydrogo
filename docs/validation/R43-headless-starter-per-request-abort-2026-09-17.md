# R43 — Headless starter çağrı-başı abort seçeneği

**Date:** 2026-09-17  
**Status:** Başarılı (yerel Node runtime)

Headless client’in public okuma ve Commerce metotları artık son parametre olarak
`HydroGoRequestOptions` kabul ediyor. `signal` böylece aynı client instance’ı
paylaşan sayfalarda yalnızca ilgili isteği iptal edebiliyor; constructor’daki
global signal ve timeout davranışları geriye dönük korunuyor.

```powershell
Push-Location examples/headless-starter
npm test
Pop-Location
```

Sonuç: **PASS** — çağrı-başı abort propagation, global/timeout abort, origin,
idempotency, typed error/request-id ve bounded response kontrolleri.
