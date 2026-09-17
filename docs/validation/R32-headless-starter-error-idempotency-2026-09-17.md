# R32 — Headless starter error and idempotency parity

**Date:** 2026-09-17  
**Status:** Başarılı (yerel TypeScript sözleşmesi)

## Değişiklik

Bağımsız `examples/headless-starter` istemcisi generated client ile aynı temel
güvenlik ve retry sözleşmesine getirildi:

- checkout idempotency key'i 16–128 görünür ASCII karakterle fail-closed
  doğrulanıyor;
- API hataları artık ham metin yerine typed `HydroAPIErrorBody` ile taşınıyor;
- `code`, kullanıcıya uygun `message` ve `x-request-id` alanları korunuyor;
- bilinen `Content-Length` değeri 2 MiB üzerindeyse gövde okunmadan reddediliyor;
- chunked/stream yanıtlar da 2 MiB byte sınırında okunuyor ve sınır aşımında
  reader iptal ediliyor;
- timeout, abort ve mevcut hata gövdesi sınırı korunuyor.

## Doğrulama

```powershell
npm run check
npm run build
```

Sonuç: **PASS**. `tsc --noEmit` ve üretim TypeScript derlemesi başarılı.

## Sınır

Bu kanıt gerçek browser/CORS/preview-token veya ödeme sağlayıcısı kabulü
değildir. Bunlar Client Edition ve gerçek ortam kapılarıdır.
