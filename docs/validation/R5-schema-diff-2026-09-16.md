# R5 — Schema/migration compatibility diff

`hydro schema diff` iki imzalı release manifestindeki migration checksum
ledger'larını karşılaştırır. Yeni migration eklenmesi uyumlu raporlanır;
mevcut migration'ın silinmesi veya checksum'ının değişmesi fail-closed olur.
Manifest imzası ve trusted public key doğrulanmadan karşılaştırma yapılmaz.

Windows doğrulaması:

```text
go test ./cmd/hydro ./internal/release
PASS
```

WSL Ubuntu doğrulaması:

```text
go test ./cmd/hydro ./internal/release
PASS
```

Çalıştırılan güncel komut `go test ./cmd/hydro ./internal/release
./internal/sdkdist` olup WSL kullanıcı `testuser` ile exit code 0 döndürdü.

Ham SQL veya veritabanı lehçesi kullanıcı çıktısına taşınmaz.
