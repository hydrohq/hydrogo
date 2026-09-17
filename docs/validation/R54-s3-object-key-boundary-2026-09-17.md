# R54 — S3 object-key and public-base boundary

**Date:** 2026-09-17  
**Status:** Başarılı (yerel contract)

S3-compatible media adapter'ı artık `Put`, `Get`, `Delete`, `PublicURL` ve
`PresignedURL` işlemlerinin tamamında aynı bounded opaque-object-key doğrulamasını
kullanıyor. Boş segment, `.`/`..`, backslash, kontrol karakteri ve aşırı uzun
anahtarlar fail-closed reddediliyor.

Public/CDN base URL'leri de credential, query veya fragment taşıyamıyor; HTTPS
zorunluluğu ve loopback-only HTTP geliştirme istisnası korunuyor.

Doğrulama: `go test ./internal/media` ve push öncesi `scripts/verify-local.ps1`.
Gerçek S3/CDN IAM, purge, retention ve failover kanıtı production kapısı olarak
açık kalır.
