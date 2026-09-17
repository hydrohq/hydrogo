# R49 — Generated client timeout parity

**Date:** 2026-09-17  
**Status:** Başarılı (yerel generator contract)

Generated `HydroClient` artık `timeoutMs` değerini starter ile aynı fail-closed
`1..120000` ms aralığında doğrular. Geçersiz değerler sessizce clamp edilmez;
constructor aşamasında görünür `RangeError` üretir. Geçerli timeout değeri
request controller’a doğrudan uygulanır.
