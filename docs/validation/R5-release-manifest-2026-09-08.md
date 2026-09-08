# R5 release manifest ve imza doğrulaması

**Tarih:** 2026-09-08  
**Roadmap:** R5 release/reproducible build kapısı  
**Durum:** Deterministic manifest ve Ed25519 verify contract'ı başarılı; gerçek CI provenance/key ceremony açık.

`internal/release` build kökündeki artifact'leri path/size/SHA-256 ile deterministically listeler; module, migration checksum, SDK/toolchain ve license inventory metadata'sını taşır. `.env`, `.key`, `.pem`, secret/credential isimli artifact'ler manifest'e alınmaz. Manifest Ed25519 ile imzalanır ve değişiklik sonrası Verify reddeder.

```text
go test ./internal/release
```

İmzalama anahtarı runtime/config/manifest içine yazılmaz. Production key ceremony, CI provenance attestation ve yeniden üretilebilir Linux/Windows build karşılaştırması release sertifikasyonunda açık kalır.
