# R47 — Ana geliştirme Compose image pinleme

**Date:** 2026-09-17  
**Status:** Başarılı (statik yapılandırma doğrulaması)

Ana geliştirme stack’i de exact sürümlere sabitlendi:

- `postgres:18.6`
- `redis:8.10.1-alpine`

Compose security lint artık `latest`, Redis major/major-minor ve PostgreSQL
yalnızca-major image tag’lerini reddediyor; patch/release tag’i olmayan yeni
servisler local verification’dan geçemiyor. Registry digest/provenance ve CVE
taraması hâlâ release pipeline kapısıdır.
