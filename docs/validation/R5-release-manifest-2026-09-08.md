# R5 release manifest ve imza doğrulaması

**Tarih:** 2026-09-08  
**Roadmap:** R5 release/reproducible build kapısı  
**Durum:** Deterministic manifest, Ed25519 verify ve manifest subject'ine bağlı builder/toolchain provenance attestation contract'ları başarılı; CI içinde aynı toolchain ile reproducible binary kontrolü eklendi. Production key ceremony, dış provenance yayınlama ve çapraz OS karşılaştırması açık.

`internal/release` build kökündeki artifact'leri path/size/SHA-256 ile deterministically listeler; module, migration checksum, SDK/toolchain ve license inventory metadata'sını taşır. `.env`, `.key`, `.pem`, secret/credential isimli artifact'ler manifest'e alınmaz. Manifest Ed25519 ile imzalanır ve değişiklik sonrası Verify reddeder.

```text
go test ./internal/release
```

İmzalama anahtarı runtime/config/manifest içine yazılmaz. CI, `hydro` binary'sini `-trimpath -buildvcs=false -ldflags=-buildid=` seçenekleriyle iki kez üretip SHA-256 değerlerini karşılaştırır. `Attest` manifest digest'ini builder, source revision ve toolchain ile imzalar; `VerifyAttestation` manifest değişikliğinde reddeder. Böylece aynı runner/toolchain üzerinde binary reproducibility ve subject bütünlüğü kanıtlanır. Production key ceremony, dış provenance yayınlama ve Linux/Windows karşılaştırması açık kalır.
