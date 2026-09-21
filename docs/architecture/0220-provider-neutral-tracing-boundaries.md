# ADR-0220: Provider-neutral tracing sınırları

## Durum

Kabul edildi — 2026-09-22.

## Karar

HTTP, repository/transaction, cache, worker, webhook ve backup işlemleri
provider-neutral span sınırları ile işaretlenir. Exporter veya sampling
zorunluluğu Core'a eklenmez; OpenTelemetry global provider deployment profili
tarafından seçilir. Span attribute'ları route, bileşen, durum ve bounded süre
ile sınırlıdır; query, body, cookie, secret ve yüksek-cardinality kimlikler
yasaktır.

## Operasyon

Compact profili exporter olmadan çalışır. Standard/Enterprise dağıtımları
OTEL SDK/provider'ı dışarıdan kurup OTLP endpoint ve sampling oranını secret
olmayan deployment config'i ile seçebilir. Collector retention ve erişim
politikasından sorumludur.
