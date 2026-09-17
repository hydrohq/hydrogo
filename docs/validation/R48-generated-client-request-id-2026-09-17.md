# R48 — Generated client request ID parity

**Date:** 2026-09-17  
**Status:** Başarılı (yerel generator contract)

Generated `HydroAPIError` artık bounded `requestId` alanını taşır. Öncelik
JSON body içindeki `request_id`, yoksa response `x-request-id` header’ıdır.
Oversize response hataları da aynı header korelasyonunu korur. Bu davranış,
headless starter’daki typed error sözleşmesiyle hizalıdır ve ham altyapı
ayrıntılarını kullanıcıya açmadan destek/observability korelasyonu sağlar.
