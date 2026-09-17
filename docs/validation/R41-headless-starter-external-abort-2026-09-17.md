# R41 — Headless starter dış abort sinyali

**Date:** 2026-09-17  
**Status:** Başarılı (yerel Node runtime)

Contract smoke testi, host uygulamanın verdiği `AbortSignal` iptalinin client
tarafındaki internal controller'a aktarıldığını ve çağrının caller reason ile
sonlandığını doğruluyor. Böylece route değişimi, kullanıcı iptali veya
framework lifecycle kapanışında istekler açık kalmıyor.

```powershell
Push-Location examples/headless-starter
npm test
Pop-Location
```

Sonuç: **PASS** — external abort propagation, timeout abort, idempotency,
typed error/request-id ve bounded response kontrolleri.
