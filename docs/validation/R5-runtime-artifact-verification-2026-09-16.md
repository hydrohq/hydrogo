# R5 runtime artifact doğrulaması — 2026-09-16

**Durum:** Başarılı (lokal/WSL)

Private runtime dağıtımının güvenli temel adımı olarak release manifestindeki
tüm artifact’ları doğrulayan `release.VerifyArtifacts` ve CLI’de `hydro release
verify --root PATH` seçeneği eklendi. İmza doğrulamasından sonra dosya boyutu,
SHA-256 özeti ve root dışına/symlink’e kaçış kontrol edilir; başarısızlıkta
runtime çalıştırılmamalıdır.

## Kanıt

- `go test ./internal/release ./cmd/hydro` başarılı.
- WSL Ubuntu/Go 1.27 üzerinde `go test ./...` başarılı.
- Testler gerçek artifact’in kabul edildiğini ve değiştirilmiş binary’nin
  reddedildiğini doğruluyor.

## Kullanım

```bash
hydro release verify --manifest release.json --public-key release.pub --root ./extracted-runtime
```

## Sınır

Bu doğrulayıcı indirme veya yetkilendirme yapmaz. Artifact broker, OAuth/device
login ve installer’lar sonraki fazlardır.
