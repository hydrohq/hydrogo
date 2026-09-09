# R5 release manifest ve imza doğrulaması

**Tarih:** 2026-09-08  
**Roadmap:** R5 release/reproducible build kapısı  
**Durum:** Deterministic manifest, Ed25519 verify ve manifest subject'ine bağlı builder/toolchain provenance attestation contract'ları başarılı; CI içinde aynı toolchain ile reproducible binary kontrolü eklendi. Production key ceremony, dış provenance yayınlama ve çapraz OS karşılaştırması açık.

`internal/release` build kökündeki artifact'leri path/size/SHA-256 ile deterministically listeler; module, migration checksum, SDK/toolchain ve license inventory metadata'sını taşır. `.env`, `.key`, `.pem`, secret/credential isimli artifact'ler manifest'e alınmaz. Manifest Ed25519 ile imzalanır ve değişiklik sonrası Verify reddeder.

```text
go test ./internal/release ./cmd/hydro
```

İmzalama anahtarı runtime/config/manifest içine yazılmaz. Operasyonel manifest doğrulaması için `hydro release verify --manifest <path> --public-key <trusted-key-path>` CLI komutu eklenmiş ve test edilmiştir; CLI, manifest içindeki anahtara tek başına güvenmez, önceden dağıtılmış trusted Ed25519 public key ile eşleşme arar. Kütüphane tarafındaki provenance doğrulaması için `VerifyAttestationWithPublicKey` aynı trusted-key kontrolünü sağlar; yalnızca embedded key ile çalışan `VerifyAttestation` self-contained imza kontrolü olarak tutulmuştur. CI, `hydro` binary'sini `-trimpath -buildvcs=false -ldflags=-buildid=` seçenekleriyle iki kez üretip SHA-256 değerlerini karşılaştırır; PostgreSQL contract ortamı yoksa CI testi artık başarısız olur ve Linux race job'ı cache/release/database/CLI paketlerini kontrol eder. `Attest` manifest digest'ini builder, source revision ve toolchain ile imzalar; `VerifyAttestation` manifest değişikliğinde reddeder. Böylece aynı runner/toolchain üzerinde binary reproducibility ve subject bütünlüğü kanıtlanır. Production key ceremony, dış provenance yayınlama ve Linux/Windows karşılaştırması açık kalır.
