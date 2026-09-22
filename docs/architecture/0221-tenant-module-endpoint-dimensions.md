# ADR-0221: Endpoint raporlarında tenant/module boyutları

## Durum

Kabul edildi — 2026-09-22.

## Karar

Endpoint telemetrisi route-normalized path istatistiklerine ek olarak en fazla
128 adet `(tenant_id,module)` boyutu tutar. Tenant yalnızca doğrulanmış tenant
scope'tan, module ise route'un bounded ilk segmentinden alınır; scope yoksa
`default` kullanılır. Query, slug ve kullanıcı kontrollü sınırsız etiketler
ölçüme girmez.

## Kanıt

`Metrics.DimensionStats`, admin JSON/CSV export ve
`TestMetricsExposeTenantAndModuleDimensions` bounded boyut sözleşmesini
doğrular.
