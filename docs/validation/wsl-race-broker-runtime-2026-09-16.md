# WSL race doğrulaması — broker/runtime — 2026-09-16

- Ubuntu WSL2, kullanıcı `testuser`
- Komut: `go test -race ./internal/sdkdist ./internal/release ./cmd/hydro`
- Sonuç: exit code 0; üç paket de `ok`, race detector uyarısı yok.

Bu test eşzamanlı memory erişimini kontrol eder; broker sağlayıcısının gerçek
yük/latency davranışını veya production network güvenliğini kanıtlamaz.
