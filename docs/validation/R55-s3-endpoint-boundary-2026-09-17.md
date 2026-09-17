# R55 — S3 endpoint URL boundary

**Date:** 2026-09-17  
**Status:** Başarılı (yerel contract)

S3-compatible endpoint URL'leri URL biçiminde verildiğinde yalnızca HTTP/HTTPS
şeması kabul ediliyor. Userinfo, query ve fragment içeren endpoint'ler sessizce
normalize edilmek yerine fail-closed reddediliyor; HTTPS endpoint `UseSSL`
seçeneğini de deterministik olarak etkinleştiriyor.

Doğrulama: `go test ./internal/media` ve push öncesi tam `verify-local` hook'u.
Gerçek sağlayıcı TLS/IAM/certificate davranışı ayrıca production kapısıdır.
