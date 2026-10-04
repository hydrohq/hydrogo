# WSL tam suite — permission contract sonrası — 2026-09-16

- Ubuntu WSL2, kullanıcı `testuser`
- Dizin: `/path/to/project`
- Komut: `go test ./... -p 1 -timeout=5m`
- Sonuç: exit code 0; tüm paketler `ok` veya `[no test files]`.

Bu çalıştırma permission preflight, generator doğrulaması ve tüm önceki SDK/
release değişikliklerini kapsar. Harici production servisleri bu testin dışındadır.
