# R34 — Headless starter client contract smoke

**Date:** 2026-09-17  
**Status:** Başarılı (yerel Node runtime)

## Kapsam

`examples/headless-starter` artık dependency-free Node `assert` ile çalışan bir
contract smoke testine sahip. Test şu davranışları gerçek derlenmiş client
üzerinde doğrular:

- geçersiz ve geçerli idempotency key sınırları;
- typed `HydroGoError`, güvenli mesaj, hata kodu ve request ID aktarımı;
- `Content-Length` olmadan gelen 2 MiB üzeri streaming response'un fail-closed
  reddi;
- normal typed news response'un başarılı çözülmesi.

## Doğrulama

```powershell
npm test
```

Sonuç: **PASS** — TypeScript build ve `PASS: headless client contract smoke`.

Test harici API, ödeme sağlayıcısı veya CORS politikasına bağlanmaz; bu nedenle
gerçek browser/production kabul kapılarının yerine geçmez.
