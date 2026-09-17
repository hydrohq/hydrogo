# R44 — Generated client ile headless starter parity

**Date:** 2026-09-17  
**Status:** Başarılı (yerel generator contract)

Project generator çıktısındaki `client/hydrogo.ts` artık headless starter ile
aynı temel sınırları taşır:

- `baseUrl` absolute HTTP(S) olmalı; embedded credential ve desteklenmeyen şema
  request öncesi reddedilir;
- `getPage` ve `listNews` çağrı-başı `{ signal }` kabul eder;
- mutation zaten `{ idempotencyKey, signal }` sözleşmesini korur.

`internal/projectgen` testi generated kaynakta bu marker'ları ve published API
route'larını doğrular. Gerçek framework/browser kabulü yine Client Edition ve
production auth topology kapısıdır.
