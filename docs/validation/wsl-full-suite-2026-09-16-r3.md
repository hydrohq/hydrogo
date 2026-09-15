# WSL tam suite — schema diff sonrası — 2026-09-16

- Dağıtım: Ubuntu WSL2
- Kullanıcı: `arda`
- Dizin: `/mnt/c/dev/cms`
- Komut: `go test ./... -p 1 -timeout=5m`
- Sonuç: exit code 0; tüm paketler `ok` veya `[no test files]`.

Bu çalıştırma signed schema compatibility diff ve ilgili release testlerini
kapsar. Harici servis ve production deployment kapıları ayrıca doğrulanmalıdır.
