# R36 — Headless starter client sınır smoke kapsamı

**Date:** 2026-09-17  
**Status:** Başarılı (yerel Node runtime)

`client-contract.mjs`, client timeout ayarının `1..120000` ms aralığında fail-closed
doğrulandığını da kontrol ediyor. Bu, yanlış yapılandırılmış frontend starter
ayarlarının istek göndermeden önce görünür olmasını sağlar.

```powershell
Push-Location examples/headless-starter
npm test
Pop-Location
```

Sonuç: **PASS** — build, idempotency, typed error/request-id, bounded response
ve timeout sınırı kontrolleri.
