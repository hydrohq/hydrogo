# R5 release manifest ve imza doğrulaması

**Tarih:** 2026-09-08  
**Roadmap:** R5 release/reproducible build kapısı  
**Durum:** Deterministic manifest ve Ed25519 verify contract'ı başarılı; CI içinde aynı toolchain ile reproducible binary kontrolü eklendi. Gerçek provenance attestation ve key ceremony açık.

`internal/release` build kökündeki artifact'leri path/size/SHA-256 ile deterministically listeler; module, migration checksum, SDK/toolchain ve license inventory metadata'sını taşır. `.env`, `.key`, `.pem`, secret/credential isimli artifact'ler manifest'e alınmaz. Manifest Ed25519 ile imzalanır ve değişiklik sonrası Verify reddeder.

```text
go test ./internal/release
```

İmzalama anahtarı runtime/config/manifest içine yazılmaz. CI, `hydro` binary'sini `-trimpath -buildvcs=false -ldflags=-buildid=` seçenekleriyle iki kez üretip SHA-256 değerlerini karşılaştırır. Böylece aynı runner/toolchain üzerinde gerçek binary reproducibility kanıtlanır. Production key ceremony, provenance attestation ve Linux/Windows karşılaştırması açık kalır.
