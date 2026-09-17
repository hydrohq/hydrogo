# R38 — Yerel doğrulama hook'una headless starter smoke kapısı

**Date:** 2026-09-17  
**Status:** Başarılı (yerel Windows)

`scripts/verify-local.ps1`, makinede `npm` bulunduğunda
`examples/headless-starter/npm test` komutunu tam Go doğrulama akışına ekler.
Node/npm olmayan minimal ortamlarda Go doğrulaması kırılmaz; kontrollü bir
uyarı verilir. CI browser-e2e job'ı ise Node kurulumu yaptığı için bu kontrolü
zorunlu çalıştırır.

Doğrulananlar: TypeScript build, idempotency key, typed API error/request-id,
2 MiB response sınırı, timeout ayar/abort davranışı.
