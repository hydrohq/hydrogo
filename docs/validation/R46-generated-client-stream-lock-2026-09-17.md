# R46 — Generated client stream lock temizliği

**Date:** 2026-09-17  
**Status:** Başarılı (yerel generator contract)

Generated `client/hydrogo.ts` bounded response reader artık normal dönüşte,
oversize hatasında ve parse/transport exception akışlarında `reader.releaseLock()`
çağrısını `finally` ile garanti ediyor. Headless starter ile aynı stream yaşam
döngüsü sözleşmesi korunuyor; response byte sınırı ve reader cancel davranışı
değişmiyor.

`internal/projectgen` testi generated kaynakta `releaseLock` marker'ını ve
mevcut published-client sözleşmesini doğruluyor.
