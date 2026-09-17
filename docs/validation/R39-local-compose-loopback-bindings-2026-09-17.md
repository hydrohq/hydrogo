# R39 — Local Compose servislerini loopback ile sınırlandırma

**Date:** 2026-09-17  
**Status:** Başarılı (statik yapılandırma doğrulaması)

`deploy/local/docker-compose.storage.yml` içindeki MinIO (API ve console) ile
Redis portları artık yalnızca `127.0.0.1` üzerinde yayınlanıyor. Ana
`compose.dev.yaml` zaten PostgreSQL ve Redis için aynı loopback sınırını
kullanıyordu. Böylece yerel geliştirme stack'i aynı ağdaki cihazlara yanlışlıkla
veritabanı, cache veya MinIO yönetim portu açmıyor.

Bu değişiklik production network policy veya gerçek firewall/ingress kanıtı
değildir; local Compose fixture'ının güvenli varsayılanıdır. Container'lar
arası erişim Compose ağı üzerinden devam eder.

Doğrulama: YAML diff kontrolü ve mevcut Go/local verification hook'ları PASS.
