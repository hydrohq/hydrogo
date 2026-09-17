# R37 — Headless starter gerçek timeout davranışı

**Date:** 2026-09-17  
**Status:** Başarılı (yerel Node runtime)

Contract smoke testi artık yalnızca timeout yapılandırma sınırlarını değil,
istek hiç tamamlanmadığında `AbortController` sinyalinin gerçekten tetiklenip
çağrının sonlandırıldığını da doğruluyor. Bu, browser uygulamasında takılı kalan
isteklerin sınırsız kaynak tüketmesini önleyen davranışın regresyon kapısıdır.

```powershell
Push-Location examples/headless-starter
npm test
Pop-Location
```

Sonuç: **PASS** — timeout abort, idempotency, typed error/request-id ve bounded
response kontrolleri.
