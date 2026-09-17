# R45 — Local Compose image tag pinleme

**Date:** 2026-09-17  
**Status:** Başarılı (statik yapılandırma doğrulaması)

Local storage Compose fixture'ındaki MinIO ve Redis image'ları mutable
`latest`/major convenience tag'leri yerine release tag'leri kullanıyor:

- `minio/minio:RELEASE.2025-09-07T16-13-09Z`
- `redis:7.4.1-alpine`

`compose-security-lint.ps1` bundan sonra tüm repo Compose fixture'larında
`:latest` image tag'ini fail-closed reddediyor. Bu, gerçek registry digest
imzalama/provenance ve vulnerability scan kapısının yerine geçmez; local
geliştirme stack'inin sürpriz image değişimini azaltır.
