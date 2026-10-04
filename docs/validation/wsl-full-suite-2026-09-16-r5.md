# WSL tam suite — artifact hygiene sonrası — 2026-09-16

- Ubuntu WSL2, kullanıcı `testuser`
- Komut: `go test ./... -p 1 -timeout=5m`
- Sonuç: exit code 0; tüm paketler `ok` veya `[no test files]`.

Bu çalıştırma release manifest artifact hygiene ve SDK runtime dağıtım
değişikliklerini kapsar. Harici imzalama anahtarı, broker, S3/CDN ve production
topolojisi bu yerel suite'in kapsamı dışındadır.
