# ADR-0218: Yapılandırılmış istek telemetrisi boyutları

## Durum

Kabul edildi — 2026-09-22.

## Karar

HTTP logları ve trace span'leri ortak, bounded boyutlar kullanır: `request_id`,
`trace_id` (varsa), normalize `route`, method, status, süre, istek/yanıt byte
sayısı ve güvenli tenant/user kimlikleri. Query string, body, cookie, e-posta,
token ve diğer PII değerleri hiçbir koşulda yazılmaz. Tenant ve user alanları
sadece doğrulanmış context değerlerinden eklenir.

## Sonuçlar

Bu sözleşme endpoint raporlarını log korelasyonu ile birleştirir ve yüksek
cardinality'yi önler. `trace_id` exporter kurulmadığında boş kalabilir;
Compact profilinde telemetry altyapısı zorunlu değildir. Log sink'i retention
ve erişim politikasını ayrıca uygulamalıdır.

## Doğrulama

`internal/observability/logging_test.go` normalize route, tenant, trace ve byte
alanlarını doğrular; mevcut log testi PII redaction'ı korur.
