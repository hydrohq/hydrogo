# R40 — Compose port binding security lint

**Date:** 2026-09-17  
**Status:** Başarılı (yerel)

`scripts/compose-security-lint.ps1` local Compose fixture'larındaki beklenen
loopback binding'lerini doğrular ve PostgreSQL, Redis veya MinIO portlarının
çıplak, `0.0.0.0` ya da başka bir host binding ile LAN'a açılmasını fail-closed
reddeder. Script,
`scripts/verify-local.ps1` akışında container hardening kontrolünün hemen
ardından çalışır.

Bu kontrol production firewall/ingress politikasının yerine geçmez; yalnızca
repo içindeki geliştirme stack'inin güvenli varsayılanını korur.
