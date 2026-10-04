# WSL tam suite — broker client sonrası — 2026-09-16

- Ubuntu WSL2, kullanıcı `testuser`
- Komut: `go test ./... -p 1 -timeout=5m`
- Sonuç: exit code 0; tüm paketler `ok` veya `[no test files]`.

Bu çalıştırma bounded broker grant istemcisi, release artifact hygiene ve önceki
SDK/runtime fazlarının tamamını kapsar. Gerçek OAuth/broker endpoint'i ve
production credential/key ceremony bu kanıtın kapsamı dışındadır.
